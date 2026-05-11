package proxy

import (
	"context"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"socks5-dpi-proxy/internal/dpi"
	"socks5-dpi-proxy/internal/pipeline"
)

type Server struct {
	listenAddr string
	configPath string
	listener   net.Listener
	pipeline   *pipeline.Engine
	detector   *dpi.AvailabilityDetector
}

func NewServer(listenAddr, configPath string) *Server {
	return &Server{
		listenAddr: listenAddr,
		configPath: configPath,
		pipeline:   pipeline.NewEngine(),
		detector:   dpi.NewAvailabilityDetector(),
	}
}

func (s *Server) Start(ctx context.Context) error {
	// Load pipeline configuration
	if err := s.pipeline.LoadConfig(s.configPath); err != nil {
		log.Printf("Warning: Failed to load pipeline config: %v", err)
	}

	listener, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	s.listener = listener

	defer listener.Close()

	log.Printf("SOCKS5 proxy listening on %s", s.listenAddr)

	// Handle shutdown
	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	// Accept connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				log.Printf("Accept error: %v", err)
				continue
			}
		}

		go s.handleConnection(ctx, conn)
	}
}

func (s *Server) handleConnection(ctx context.Context, clientConn net.Conn) {
	defer clientConn.Close()

	clientConn.SetDeadline(time.Now().Add(30 * time.Second))

	// Handle SOCKS5 authentication
	if err := handleAuthentication(clientConn); err != nil {
		log.Printf("Authentication failed: %v", err)
		return
	}

	// Parse SOCKS5 request
	req, err := parseRequest(clientConn)
	if err != nil {
		log.Printf("Failed to parse request: %v", err)
		sendReply(clientConn, RepGeneralFailure, "", 0)
		return
	}

	// Check if domain is available directly (optimization)
	domain := s.detector.ExtractDomain(req.DstAddr)
	shouldUsePipeline := s.pipeline.ShouldUsePipeline(domain, req.DstPort)

	// Start availability check in background if not cached
	go func() {
		s.detector.CheckDomain(ctx, domain)
	}()

	var pipe *pipeline.Pipeline
	if shouldUsePipeline {
		pipe = s.pipeline.CreatePipeline(domain, req.DstPort)
		log.Printf("Using DPI bypass pipeline for %s:%d", domain, req.DstPort)
	} else {
		log.Printf("Using direct connection for %s:%d (domain available)", domain, req.DstPort)
	}

	// Connect to destination
	targetConn, err := s.establishConnection(ctx, req, pipe)
	if err != nil {
		log.Printf("Failed to connect to %s:%d: %v", req.DstAddr, req.DstPort, err)
		sendReply(clientConn, RepHostUnreachable, "", 0)
		return
	}
	defer targetConn.Close()

	// Send success reply
	if err := sendReply(clientConn, RepSuccess, "", 0); err != nil {
		log.Printf("Failed to send reply: %v", err)
		return
	}

	// Remove deadline for data transfer
	clientConn.SetDeadline(time.Time{})
	targetConn.SetDeadline(time.Time{})

	// Start bidirectional data relay with pipeline processing
	var wg sync.WaitGroup
	wg.Add(2)

	// Client to target
	go func() {
		defer wg.Done()
		s.relayData(ctx, clientConn, targetConn, pipe, true)
	}()

	// Target to client
	go func() {
		defer wg.Done()
		s.relayData(ctx, targetConn, clientConn, pipe, false)
	}()

	wg.Wait()
}

func (s *Server) establishConnection(ctx context.Context, req *SOCKS5Request, pipe *pipeline.Pipeline) (net.Conn, error) {
	// Apply pre-connection modifications if any
	if pipe != nil {
		// Convert to pipeline request type
		pipeReq := &pipeline.SOCKS5Request{
			Version:  req.Version,
			Command:  req.Command,
			Reserved: req.Reserved,
			AddrType: req.AddrType,
			DstAddr:  req.DstAddr,
			DstPort:  req.DstPort,
		}

		if modifiedReq := pipe.PreConnection(pipeReq); modifiedReq != nil {
			// Convert back to proxy request type
			req = &SOCKS5Request{
				Version:  modifiedReq.Version,
				Command:  modifiedReq.Command,
				Reserved: modifiedReq.Reserved,
				AddrType: modifiedReq.AddrType,
				DstAddr:  modifiedReq.DstAddr,
				DstPort:  modifiedReq.DstPort,
			}
		}
	}

	// Try direct connection first
	conn, err := connectToDestination(req)
	if err == nil {
		return conn, nil
	}

	// If direct connection fails and we have a pipeline, try with pipeline
	if pipe != nil {
		log.Printf("Direct connection failed, retrying with DPI bypass: %v", err)
		return connectToDestination(req)
	}

	return nil, err
}

func (s *Server) relayData(ctx context.Context, src, dst net.Conn, pipe *pipeline.Pipeline, isClientToTarget bool) {
	buf := make([]byte, 32*1024) // 32KB buffer

	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := src.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Read error: %v", err)
				}
				return
			}

			if n == 0 {
				continue
			}

			data := buf[:n]

			// Apply pipeline modifications
			if pipe != nil {
				if isClientToTarget {
					data = pipe.ProcessOutbound(data)
				} else {
					data = pipe.ProcessInbound(data)
				}
			}

			// Write processed data
			_, err = dst.Write(data)
			if err != nil {
				log.Printf("Write error: %v", err)
				return
			}
		}
	}
}

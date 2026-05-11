package proxy

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	SOCKS5Version = 0x05

	// Authentication methods
	AuthNoAuth       = 0x00
	AuthGSSAPI       = 0x01
	AuthUserPass     = 0x02
	AuthNoAcceptable = 0xFF

	// Commands
	CmdConnect      = 0x01
	CmdBind         = 0x02
	CmdUDPAssociate = 0x03

	// Address types
	AddrIPv4   = 0x01
	AddrDomain = 0x03
	AddrIPv6   = 0x04

	// Reply codes
	RepSuccess              = 0x00
	RepGeneralFailure       = 0x01
	RepConnectionNotAllowed = 0x02
	RepNetworkUnreachable   = 0x03
	RepHostUnreachable      = 0x04
	RepConnectionRefused    = 0x05
	RepTTLExpired           = 0x06
	RepCommandNotSupported  = 0x07
	RepAddressNotSupported  = 0x08
)

var (
	ErrInvalidVersion    = errors.New("invalid SOCKS version")
	ErrInvalidCommand    = errors.New("invalid command")
	ErrInvalidAuthMethod = errors.New("invalid authentication method")
	ErrConnectionFailed  = errors.New("connection failed")
)

type SOCKS5Request struct {
	Version  byte
	Command  byte
	Reserved byte
	AddrType byte
	DstAddr  string
	DstPort  uint16
}

type SOCKS5Reply struct {
	Version  byte
	Reply    byte
	Reserved byte
	AddrType byte
	BndAddr  string
	BndPort  uint16
}

func handleAuthentication(conn net.Conn) error {
	// Read authentication request
	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}

	if buf[0] != SOCKS5Version {
		return ErrInvalidVersion
	}

	nMethods := int(buf[1])
	if nMethods == 0 {
		return ErrInvalidAuthMethod
	}

	// Read methods
	methods := make([]byte, nMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return err
	}

	// Check if no authentication is supported
	hasNoAuth := false
	for _, method := range methods {
		if method == AuthNoAuth {
			hasNoAuth = true
			break
		}
	}

	if !hasNoAuth {
		// Send no acceptable methods
		_, err := conn.Write([]byte{SOCKS5Version, AuthNoAcceptable})
		return err
	}

	// Send no authentication method selection
	_, err := conn.Write([]byte{SOCKS5Version, AuthNoAuth})
	return err
}

func parseRequest(conn net.Conn) (*SOCKS5Request, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}

	if header[0] != SOCKS5Version {
		return nil, ErrInvalidVersion
	}

	if header[1] != CmdConnect {
		return nil, ErrInvalidCommand
	}

	req := &SOCKS5Request{
		Version:  header[0],
		Command:  header[1],
		Reserved: header[2],
		AddrType: header[3],
	}

	// Parse destination address based on address type
	switch req.AddrType {
	case AddrIPv4:
		addr := make([]byte, 4)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return nil, err
		}
		req.DstAddr = net.IP(addr).String()

	case AddrDomain:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return nil, err
		}

		domainLen := int(lenBuf[0])
		domain := make([]byte, domainLen)
		if _, err := io.ReadFull(conn, domain); err != nil {
			return nil, err
		}
		req.DstAddr = string(domain)

	case AddrIPv6:
		addr := make([]byte, 16)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return nil, err
		}
		req.DstAddr = net.IP(addr).String()

	default:
		return nil, errors.New("unsupported address type")
	}

	// Read destination port
	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return nil, err
	}
	req.DstPort = binary.BigEndian.Uint16(portBuf)

	return req, nil
}

func sendReply(conn net.Conn, reply byte, bindAddr string, bindPort uint16) error {
	var replyMsg []byte

	if bindAddr != "" {
		ip := net.ParseIP(bindAddr)
		if ip != nil {
			if ip.To4() != nil {
				// IPv4 response
				replyMsg = make([]byte, 10) // 4 + 4 + 2
				replyMsg[3] = AddrIPv4
				copy(replyMsg[4:8], ip.To4())
			} else {
				// IPv6 response
				replyMsg = make([]byte, 22) // 4 + 16 + 2
				replyMsg[3] = AddrIPv6
				copy(replyMsg[4:20], ip.To16())
			}
		} else {
			// Default IPv4
			replyMsg = make([]byte, 10)
			replyMsg[3] = AddrIPv4
		}
	} else {
		// Default IPv4
		replyMsg = make([]byte, 10)
		replyMsg[3] = AddrIPv4
	}

	// Set standard fields
	replyMsg[0] = SOCKS5Version
	replyMsg[1] = reply
	replyMsg[2] = 0x00 // Reserved

	// Set port (always at the end)
	if bindPort > 0 {
		binary.BigEndian.PutUint16(replyMsg[len(replyMsg)-2:], bindPort)
	}

	_, err := conn.Write(replyMsg)
	return err
}

func connectToDestination(req *SOCKS5Request) (net.Conn, error) {
	target := net.JoinHostPort(req.DstAddr, fmt.Sprintf("%d", req.DstPort))

	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	// Try IPv6 first if the destination is IPv6
	ip := net.ParseIP(req.DstAddr)
	if ip != nil && ip.To4() == nil {
		// IPv6 address - try IPv6 first, then fallback to IPv4
		if conn, err := dialer.Dial("tcp6", target); err == nil {
			return conn, nil
		}
		return dialer.Dial("tcp4", target)
	}

	// For domain names or IPv4, try dual-stack
	return dialer.Dial("tcp", target)
}

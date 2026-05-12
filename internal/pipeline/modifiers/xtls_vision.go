package modifiers

import (
	"crypto/rand"
	"io"
	"net"
	"sync"
)

// TrafficState represents the state of XTLS Vision flow control
type TrafficState struct {
	InboundState  int
	OutboundState int
	PaddingLeft   int
	CommandSent   bool
	mu            sync.RWMutex
}

const (
	// Traffic states
	StateInitial = iota
	StateCommand
	StateTransfer
	StatePadding
	StateFinished
)

// NewTrafficState creates a new traffic state
func NewTrafficState() *TrafficState {
	return &TrafficState{
		InboundState:  StateInitial,
		OutboundState: StateInitial,
		PaddingLeft:   0,
		CommandSent:   false,
	}
}

// SetInboundState sets the inbound traffic state
func (ts *TrafficState) SetInboundState(state int) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.InboundState = state
}

// SetOutboundState sets the outbound traffic state
func (ts *TrafficState) SetOutboundState(state int) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.OutboundState = state
}

// GetInboundState returns the inbound traffic state
func (ts *TrafficState) GetInboundState() int {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.InboundState
}

// GetOutboundState returns the outbound traffic state
func (ts *TrafficState) GetOutboundState() int {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.OutboundState
}

// SetPaddingLeft sets the remaining padding
func (ts *TrafficState) SetPaddingLeft(padding int) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.PaddingLeft = padding
}

// GetPaddingLeft returns the remaining padding
func (ts *TrafficState) GetPaddingLeft() int {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.PaddingLeft
}

// SetCommandSent marks if command has been sent
func (ts *TrafficState) SetCommandSent(sent bool) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.CommandSent = sent
}

// IsCommandSent returns if command has been sent
func (ts *TrafficState) IsCommandSent() bool {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.CommandSent
}

// VisionReader implements XTLS Vision flow control for reading
type VisionReader struct {
	conn   net.Conn
	state  *TrafficState
	buffer []byte
	pos    int
	mu     sync.RWMutex
}

// NewVisionReader creates a new VisionReader
func NewVisionReader(conn net.Conn) *VisionReader {
	return &VisionReader{
		conn:   conn,
		state:  NewTrafficState(),
		buffer: make([]byte, 0, 4096),
		pos:    0,
	}
}

// Read implements io.Reader with Vision flow control
func (vr *VisionReader) Read(p []byte) (int, error) {
	vr.mu.Lock()
	defer vr.mu.Unlock()

	// If we have data in buffer, return from buffer first
	if vr.pos < len(vr.buffer) {
		n := copy(p, vr.buffer[vr.pos:])
		vr.pos += n
		return n, nil
	}

	// Read from connection
	n, err := vr.conn.Read(p)
	if err != nil {
		return n, err
	}

	// Process Vision flow logic
	vr.processVisionRead(p[:n])

	return n, nil
}

// processVisionRead processes incoming data according to Vision flow
func (vr *VisionReader) processVisionRead(data []byte) {
	state := vr.state.GetInboundState()

	switch state {
	case StateInitial:
		// Initial state, look for command
		if len(data) >= 1 {
			vr.state.SetInboundState(StateCommand)
			vr.state.SetCommandSent(true)
		}
	case StateCommand:
		// Command received, switch to transfer
		vr.state.SetInboundState(StateTransfer)
	case StateTransfer:
		// Transfer state, continue processing
		// Check if we need to handle padding
		if vr.state.GetPaddingLeft() > 0 {
			vr.handlePadding(data)
		}
	case StatePadding:
		// Padding state
		vr.state.SetInboundState(StateFinished)
	}
}

// handlePadding handles padding data
func (vr *VisionReader) handlePadding(data []byte) {
	paddingLeft := vr.state.GetPaddingLeft()
	if len(data) >= paddingLeft {
		// Remove padding
		vr.buffer = append(vr.buffer, data[paddingLeft:]...)
		vr.state.SetPaddingLeft(0)
	} else {
		// Still padding left
		vr.state.SetPaddingLeft(paddingLeft - len(data))
	}
}

// WriteTo implements io.WriterTo for efficient copying
func (vr *VisionReader) WriteTo(w io.Writer) (int64, error) {
	// Use splice/copy optimization if possible
	if vr.state.GetInboundState() == StateTransfer {
		return vr.spliceTo(w)
	}

	// Fallback to regular copy
	buf := make([]byte, 4096)
	return io.CopyBuffer(w, vr, buf)
}

// spliceTo attempts to use splice for zero-copy operations
func (vr *VisionReader) spliceTo(w io.Writer) (int64, error) {
	// Check if target is a TCP connection with splice capability
	if dstConn, ok := w.(interface{ Splice(net.Conn) (int64, error) }); ok {
		if srcTCP, ok := vr.conn.(*net.TCPConn); ok {
			return dstConn.Splice(srcTCP)
		}
	}

	// Fallback to regular copy with buffer
	buf := make([]byte, 4096)
	return io.CopyBuffer(w, vr, buf)
}

// VisionWriter implements XTLS Vision flow control for writing
type VisionWriter struct {
	conn  net.Conn
	state *TrafficState
	mu    sync.RWMutex
}

// NewVisionWriter creates a new VisionWriter
func NewVisionWriter(conn net.Conn) *VisionWriter {
	return &VisionWriter{
		conn:  conn,
		state: NewTrafficState(),
	}
}

// Write implements io.Writer with Vision flow control
func (vw *VisionWriter) Write(p []byte) (int, error) {
	vw.mu.Lock()
	defer vw.mu.Unlock()

	state := vw.state.GetOutboundState()

	switch state {
	case StateInitial:
		// Write command first
		if err := vw.writeCommand(); err != nil {
			return 0, err
		}
		vw.state.SetOutboundState(StateTransfer)
		fallthrough
	case StateTransfer:
		// Write data with potential padding
		return vw.writeWithPadding(p)
	default:
		// Direct write for other states
		return vw.conn.Write(p)
	}
}

// writeCommand writes the initial VLESS command
func (vw *VisionWriter) writeCommand() error {
	// VLESS command structure
	// Version (1 byte) + Command (1 byte) + Protocol (1 byte) + Reserved (2 bytes)
	cmd := []byte{0x00, 0x01, 0x01, 0x00, 0x00}

	_, err := vw.conn.Write(cmd)
	return err
}

// writeWithPadding writes data with padding according to Vision flow
func (vw *VisionWriter) writeWithPadding(p []byte) (int, error) {
	// Add padding if needed
	padding := vw.calculatePadding(len(p))

	if padding > 0 {
		// Write data + padding
		data := append(p, make([]byte, padding)...)
		_, err := vw.conn.Write(data)
		if err != nil {
			return 0, err
		}
		vw.state.SetPaddingLeft(padding)
		return len(p), nil
	}

	// Direct write
	return vw.conn.Write(p)
}

// calculatePadding calculates required padding
func (vw *VisionWriter) calculatePadding(dataLen int) int {
	// Vision padding strategy
	// Pad to multiple of 16 bytes for better obfuscation
	remainder := dataLen % 16
	if remainder == 0 {
		return 0
	}
	return 16 - remainder
}

// AddNoise adds random noise for additional obfuscation
func AddNoise(data []byte) []byte {
	// Add small amount of random noise
	noise := make([]byte, 4)
	rand.Read(noise)

	result := make([]byte, len(data)+4)
	copy(result, data)
	copy(result[len(data):], noise)

	return result
}

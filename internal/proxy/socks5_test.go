package proxy

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"
	"time"
)

func TestHandleAuthentication(t *testing.T) {
	tests := []struct {
		name          string
		input         []byte
		expectError   bool
		expectedReply []byte
	}{
		{
			name:          "Valid SOCKS5 with no-auth",
			input:         []byte{0x05, 0x01, 0x00},
			expectError:   false,
			expectedReply: []byte{0x05, 0x00},
		},
		{
			name:          "Valid SOCKS5 with no-auth among others",
			input:         []byte{0x05, 0x02, 0x00, 0x01},
			expectError:   false,
			expectedReply: []byte{0x05, 0x00},
		},
		{
			name:          "Invalid SOCKS version",
			input:         []byte{0x04, 0x01, 0x00},
			expectError:   true,
			expectedReply: nil,
		},
		{
			name:          "No authentication methods",
			input:         []byte{0x05, 0x00},
			expectError:   true,
			expectedReply: nil,
		},
		{
			name:          "No acceptable methods",
			input:         []byte{0x05, 0x01, 0x01},
			expectError:   false,
			expectedReply: []byte{0x05, 0xFF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &mockConn{data: tt.input}
			err := handleAuthentication(conn)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && !bytes.Equal(conn.written, tt.expectedReply) {
				t.Errorf("Expected reply %v, got %v", tt.expectedReply, conn.written)
			}
		})
	}
}

func TestParseRequest(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expectError bool
		expectedReq *SOCKS5Request
	}{
		{
			name: "Valid IPv4 request",
			input: []byte{
				0x05, 0x01, 0x00, 0x01, // Version, Command, Reserved, AddrType
				0x08, 0x08, 0x08, 0x08, // IP: 8.8.8.8
				0x1F, 0x90, // Port: 8080
			},
			expectError: false,
			expectedReq: &SOCKS5Request{
				Version:  0x05,
				Command:  0x01,
				Reserved: 0x00,
				AddrType: 0x01,
				DstAddr:  "8.8.8.8",
				DstPort:  8080,
			},
		},
		{
			name: "Valid domain request",
			input: []byte{
				0x05, 0x01, 0x00, 0x03, // Version, Command, Reserved, AddrType
				0x0B,                                                  // Domain length: 11
				'e', 'x', 'a', 'm', 'p', 'l', 'e', '.', 'c', 'o', 'm', // example.com
				0x00, 0x50, // Port: 80 (big-endian)
			},
			expectError: false,
			expectedReq: &SOCKS5Request{
				Version:  0x05,
				Command:  0x01,
				Reserved: 0x00,
				AddrType: 0x03,
				DstAddr:  "example.com",
				DstPort:  80,
			},
		},
		{
			name: "Valid IPv6 request",
			input: []byte{
				0x05, 0x01, 0x00, 0x04, // Version, Command, Reserved, AddrType
				0x20, 0x01, 0x0D, 0xB8, // IPv6: 2001:db8::
				0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x01,
				0x1F, 0x90, // Port: 8080
			},
			expectError: false,
			expectedReq: &SOCKS5Request{
				Version:  0x05,
				Command:  0x01,
				Reserved: 0x00,
				AddrType: 0x04,
				DstAddr:  "2001:db8::1",
				DstPort:  8080,
			},
		},
		{
			name: "Invalid SOCKS version",
			input: []byte{
				0x04, 0x01, 0x00, 0x01,
				0x08, 0x08, 0x08, 0x08,
				0x1F, 0x90,
			},
			expectError: true,
			expectedReq: nil,
		},
		{
			name: "Unsupported command",
			input: []byte{
				0x05, 0x02, 0x00, 0x01, // BIND command
				0x08, 0x08, 0x08, 0x08,
				0x1F, 0x90,
			},
			expectError: true,
			expectedReq: nil,
		},
		{
			name: "Unsupported address type",
			input: []byte{
				0x05, 0x01, 0x00, 0x02, // Unsupported address type
				0x08, 0x08, // Some data
				0x1F, 0x90,
			},
			expectError: true,
			expectedReq: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &mockConn{data: tt.input}
			req, err := parseRequest(conn)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError {
				if req == nil {
					t.Errorf("Expected request but got nil")
					return
				}
				if req.Version != tt.expectedReq.Version ||
					req.Command != tt.expectedReq.Command ||
					req.Reserved != tt.expectedReq.Reserved ||
					req.AddrType != tt.expectedReq.AddrType ||
					req.DstAddr != tt.expectedReq.DstAddr ||
					req.DstPort != tt.expectedReq.DstPort {
					t.Errorf("Request mismatch:\nExpected: %+v\nGot:      %+v", tt.expectedReq, req)
				}
			}
		})
	}
}

func TestSendReply(t *testing.T) {
	tests := []struct {
		name         string
		reply        byte
		bindAddr     string
		bindPort     uint16
		expectedIPv4 bool
		expectedIPv6 bool
	}{
		{
			name:         "Success reply with IPv4",
			reply:        RepSuccess,
			bindAddr:     "192.168.1.100",
			bindPort:     1080,
			expectedIPv4: true,
			expectedIPv6: false,
		},
		{
			name:         "Success reply with IPv6",
			reply:        RepSuccess,
			bindAddr:     "2001:db8::1",
			bindPort:     1080,
			expectedIPv4: false,
			expectedIPv6: true,
		},
		{
			name:         "Success reply no address",
			reply:        RepSuccess,
			bindAddr:     "",
			bindPort:     0,
			expectedIPv4: true, // Default to IPv4
			expectedIPv6: false,
		},
		{
			name:         "General failure",
			reply:        RepGeneralFailure,
			bindAddr:     "",
			bindPort:     0,
			expectedIPv4: true,
			expectedIPv6: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &mockConn{}
			err := sendReply(conn, tt.reply, tt.bindAddr, tt.bindPort)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(conn.written) < 4 {
				t.Errorf("Reply too short: %d bytes", len(conn.written))
				return
			}

			// Check basic SOCKS5 reply format
			if conn.written[0] != SOCKS5Version {
				t.Errorf("Expected version %d, got %d", SOCKS5Version, conn.written[0])
			}
			if conn.written[1] != tt.reply {
				t.Errorf("Expected reply %d, got %d", tt.reply, conn.written[1])
			}
			if conn.written[2] != 0x00 {
				t.Errorf("Expected reserved 0x00, got %d", conn.written[2])
			}

			// Check address type
			if tt.expectedIPv4 && conn.written[3] != AddrIPv4 {
				t.Errorf("Expected IPv4 address type %d, got %d", AddrIPv4, conn.written[3])
			}
			if tt.expectedIPv6 && conn.written[3] != AddrIPv6 {
				t.Errorf("Expected IPv6 address type %d, got %d", AddrIPv6, conn.written[3])
			}

			// Check port
			if tt.bindPort > 0 {
				var expectedLen int
				if tt.expectedIPv6 {
					expectedLen = 22 // 4 + 16 + 2
				} else {
					expectedLen = 10 // 4 + 4 + 2
				}

				if len(conn.written) != expectedLen {
					t.Errorf("Expected reply length %d, got %d", expectedLen, len(conn.written))
				}

				port := binary.BigEndian.Uint16(conn.written[len(conn.written)-2:])
				if port != tt.bindPort {
					t.Errorf("Expected port %d, got %d", tt.bindPort, port)
				}
			}
		})
	}
}

func TestConnectToDestination(t *testing.T) {
	tests := []struct {
		name        string
		req         *SOCKS5Request
		expectError bool
	}{
		{
			name: "Valid IPv4 request",
			req: &SOCKS5Request{
				DstAddr: "1.1.1.1",
				DstPort: 53,
			},
			expectError: false, // Should connect to Cloudflare DNS
		},
		{
			name: "Valid domain request",
			req: &SOCKS5Request{
				DstAddr: "google.com",
				DstPort: 80,
			},
			expectError: false, // Should connect to Google
		},
		{
			name: "Valid IPv6 request",
			req: &SOCKS5Request{
				DstAddr: "2001:4860:4860::8888",
				DstPort: 53,
			},
			expectError: false, // Should connect to Google IPv6 DNS
		},
		{
			name: "Invalid address",
			req: &SOCKS5Request{
				DstAddr: "invalid.address.example",
				DstPort: 80,
			},
			expectError: true, // Should fail to connect
		},
		{
			name: "Unreachable port",
			req: &SOCKS5Request{
				DstAddr: "127.0.0.1",
				DstPort: 65432, // Unlikely to be listening (fits in uint16)
			},
			expectError: true, // Should fail to connect
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := connectToDestination(tt.req)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
				if conn != nil {
					conn.Close()
				}
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectError && conn != nil {
				conn.Close()
			}
		})
	}
}

// Mock connection for testing
type mockConn struct {
	data    []byte
	written []byte
	pos     int
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	if m.pos >= len(m.data) {
		return 0, nil // EOF
	}
	n = copy(b, m.data[m.pos:])
	m.pos += n
	return n, nil
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	m.written = append(m.written, b...)
	return len(b), nil
}

func (m *mockConn) Close() error { return nil }
func (m *mockConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1080}
}
func (m *mockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
}
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

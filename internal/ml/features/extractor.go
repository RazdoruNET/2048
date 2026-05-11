package features

import (
	"bytes"
	"encoding/binary"
	"strings"
	"time"
)

// FeatureExtractor extracts features from network traffic
type FeatureExtractor interface {
	ExtractPacketFeatures(data []byte) []float64
	ExtractFlowFeatures(flow []byte) []float64
	ExtractTimingFeatures(timestamps []time.Time) []float64
	ExtractHTTPFeatures(data []byte) []float64
	ExtractTLSFeatures(data []byte) []float64
}

// PacketFeatures represents features extracted from a single packet
type PacketFeatures struct {
	Size        float64
	Protocol    float64
	Flags       float64
	Sequence    float64
	Ack         float64
	Window      float64
	Urgent      float64
	HasPayload  float64
	PayloadSize float64
	HeaderSize  float64
	Timestamp   float64
}

// FlowFeatures represents features extracted from a flow
type FlowFeatures struct {
	TotalBytes     float64
	PacketCount    float64
	AvgPacketSize  float64
	MaxPacketSize  float64
	MinPacketSize  float64
	Duration       float64
	BytesPerSecond float64
	PacketsPerSec  float64
	Direction      float64
	InterArrival   float64
}

// HTTPFeatures represents HTTP-specific features
type HTTPFeatures struct {
	Method        float64
	HasUserAgent  float64
	HasCookies    float64
	HeaderCount   float64
	ContentLength float64
	HasHTTPS      float64
	HostLength    float64
	PathLength    float64
	IsGET         float64
	IsPOST        float64
}

// TLSFeatures represents TLS-specific features
type TLSFeatures struct {
	Version       float64
	CipherSuite   float64
	HasSNI        float64
	SNILength     float64
	CertLength    float64
	HandshakeTime float64
	IsTLS13       float64
	HasALPN       float64
}

// DefaultFeatureExtractor implements FeatureExtractor
type DefaultFeatureExtractor struct{}

// NewDefaultFeatureExtractor creates a new feature extractor
func NewDefaultFeatureExtractor() *DefaultFeatureExtractor {
	return &DefaultFeatureExtractor{}
}

// ExtractPacketFeatures extracts features from a single packet
func (e *DefaultFeatureExtractor) ExtractPacketFeatures(data []byte) []float64 {
	features := e.parsePacketFeatures(data)

	return []float64{
		features.Size,
		features.Protocol,
		features.Flags,
		features.Sequence,
		features.Ack,
		features.Window,
		features.Urgent,
		features.HasPayload,
		features.PayloadSize,
		features.HeaderSize,
		features.Timestamp,
	}
}

// ExtractFlowFeatures extracts features from a flow
func (e *DefaultFeatureExtractor) ExtractFlowFeatures(flow []byte) []float64 {
	features := e.parseFlowFeatures(flow)

	return []float64{
		features.TotalBytes,
		features.PacketCount,
		features.AvgPacketSize,
		features.MaxPacketSize,
		features.MinPacketSize,
		features.Duration,
		features.BytesPerSecond,
		features.PacketsPerSec,
		features.Direction,
		features.InterArrival,
	}
}

// ExtractTimingFeatures extracts timing-based features
func (e *DefaultFeatureExtractor) ExtractTimingFeatures(timestamps []time.Time) []float64 {
	if len(timestamps) < 2 {
		return make([]float64, 10) // Return empty features
	}

	var intervals []float64
	for i := 1; i < len(timestamps); i++ {
		interval := timestamps[i].Sub(timestamps[i-1]).Seconds()
		intervals = append(intervals, interval)
	}

	// Calculate statistics
	var sum, sumSq, min, max float64
	min = intervals[0]
	max = intervals[0]

	for _, interval := range intervals {
		sum += interval
		sumSq += interval * interval
		if interval < min {
			min = interval
		}
		if interval > max {
			max = interval
		}
	}

	mean := sum / float64(len(intervals))
	variance := (sumSq / float64(len(intervals))) - (mean * mean)

	return []float64{
		mean,
		variance,
		min,
		max,
		float64(len(intervals)),
		sum,
		sumSq,
		mean / (max + 0.0001),       // Coefficient of variation
		intervals[len(intervals)-1], // Last interval
		intervals[0],                // First interval
	}
}

// ExtractHTTPFeatures extracts HTTP-specific features
func (e *DefaultFeatureExtractor) ExtractHTTPFeatures(data []byte) []float64 {
	features := e.parseHTTPFeatures(data)

	return []float64{
		features.Method,
		features.HasUserAgent,
		features.HasCookies,
		features.HeaderCount,
		features.ContentLength,
		features.HasHTTPS,
		features.HostLength,
		features.PathLength,
		features.IsGET,
		features.IsPOST,
	}
}

// ExtractTLSFeatures extracts TLS-specific features
func (e *DefaultFeatureExtractor) ExtractTLSFeatures(data []byte) []float64 {
	features := e.parseTLSFeatures(data)

	return []float64{
		features.Version,
		features.CipherSuite,
		features.HasSNI,
		features.SNILength,
		features.CertLength,
		features.HandshakeTime,
		features.IsTLS13,
		features.HasALPN,
	}
}

// parsePacketFeatures parses packet-level features
func (e *DefaultFeatureExtractor) parsePacketFeatures(data []byte) PacketFeatures {
	features := PacketFeatures{
		Size:      float64(len(data)),
		Timestamp: float64(time.Now().UnixNano()) / 1e9,
	}

	if len(data) >= 20 {
		// Parse IP header (simplified)
		protocol := float64(data[9])
		features.Protocol = protocol

		// Parse TCP header (simplified)
		if protocol == 6 && len(data) >= 40 {
			flags := float64(data[47])
			features.Flags = flags
			features.Sequence = float64(binary.BigEndian.Uint32(data[42:46]))
			features.Ack = float64(binary.BigEndian.Uint32(data[46:50]))
			features.Window = float64(binary.BigEndian.Uint16(data[54:56]))
			features.Urgent = float64(data[51])
		}
	}

	// Calculate payload and header sizes
	if len(data) >= 40 {
		features.HeaderSize = 40 // IP + TCP headers
		features.PayloadSize = float64(len(data) - 40)
		features.HasPayload = 1.0
	} else {
		features.HeaderSize = float64(len(data))
		features.PayloadSize = 0
		features.HasPayload = 0
	}

	return features
}

// parseFlowFeatures parses flow-level features
func (e *DefaultFeatureExtractor) parseFlowFeatures(flow []byte) FlowFeatures {
	features := FlowFeatures{
		TotalBytes:  float64(len(flow)),
		PacketCount: 1, // Simplified - would need packet list for real implementation
	}

	features.AvgPacketSize = features.TotalBytes / features.PacketCount
	features.MaxPacketSize = features.TotalBytes
	features.MinPacketSize = features.TotalBytes
	features.BytesPerSecond = features.TotalBytes
	features.PacketsPerSec = features.PacketCount

	return features
}

// parseHTTPFeatures parses HTTP-specific features
func (e *DefaultFeatureExtractor) parseHTTPFeatures(data []byte) HTTPFeatures {
	features := HTTPFeatures{}

	// Convert to string for parsing
	dataStr := string(data)

	// Parse HTTP method
	lines := strings.Split(dataStr, "\n")
	if len(lines) > 0 {
		fields := strings.Fields(lines[0])
		if len(fields) > 0 {
			switch strings.ToUpper(fields[0]) {
			case "GET":
				features.Method = 1 // GET
				features.IsGET = 1
			case "POST":
				features.Method = 2 // POST
				features.IsPOST = 1
			case "HEAD":
				features.Method = 3 // HEAD
			case "DELETE":
				features.Method = 4 // DELETE
			case "PUT":
				features.Method = 5 // PUT
			}
		}
	}

	// Count headers and check for specific ones
	headerCount := 0
	hasUserAgent := 0.0
	hasCookies := 0.0
	contentLength := 0.0
	hostLength := 0.0
	pathLength := 0.0

	for _, line := range lines {
		if strings.Contains(line, "User-Agent:") {
			hasUserAgent = 1
		}
		if strings.Contains(line, "Cookie:") {
			hasCookies = 1
		}
		if strings.Contains(line, "Content-Length:") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				contentLength = 1 // Normalize
			}
		}
		if strings.Contains(line, "Host:") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				hostLength = float64(len(parts[1]))
			}
		}
		if strings.TrimSpace(line) != "" {
			headerCount++
		}
	}

	features.HeaderCount = float64(headerCount)
	features.HasUserAgent = hasUserAgent
	features.HasCookies = hasCookies
	features.ContentLength = contentLength
	features.HostLength = hostLength
	features.PathLength = pathLength
	features.HasHTTPS = 1.0 // Default to HTTPS for modern traffic

	return features
}

// parseTLSFeatures parses TLS-specific features
func (e *DefaultFeatureExtractor) parseTLSFeatures(data []byte) TLSFeatures {
	features := TLSFeatures{}

	if len(data) < 5 {
		return features
	}

	// Check for TLS handshake (simplified)
	if data[0] == 0x16 { // TLS handshake
		features.HasSNI = 1

		// Extract version (simplified)
		if len(data) >= 3 {
			version := binary.BigEndian.Uint16(data[1:3])
			features.Version = float64(version)

			// Check for TLS 1.3
			if version == 0x0304 {
				features.IsTLS13 = 1
			}
		}

		// Look for SNI (simplified parsing)
		for i := 5; i < len(data)-10; i++ {
			if data[i] == 0x00 && bytes.Contains(data[i:i+10], []byte("google")) {
				features.SNILength = 6 // Simplified
				break
			}
		}
	}

	return features
}

package pipeline

import (
	"net"
	"sync"
	"time"
)

// ModifiedConnectionV2 wraps a connection with enhanced fragmentation support
type ModifiedConnectionV2 struct {
	net.Conn
	Modifier   ModifierV2
	Request    *SOCKS5Request
	Config     *ConnectionConfig
	writeStats *WriteStats
	mu         sync.RWMutex
}

// WriteStats tracks write performance metrics
type WriteStats struct {
	TotalWrites      uint64        `json:"total_writes"`
	TotalBytes       uint64        `json:"total_bytes"`
	FragmentedWrites uint64        `json:"fragmented_writes"`
	AverageLatency   time.Duration `json:"average_latency"`
	ErrorCount       uint64        `json:"error_count"`
	LastWriteTime    time.Time     `json:"last_write_time"`
	mu               sync.RWMutex
}

// NewModifiedConnectionV2 creates a new enhanced modified connection
func NewModifiedConnectionV2(conn net.Conn, modifier ModifierV2, request *SOCKS5Request, config *ConnectionConfig) *ModifiedConnectionV2 {
	if config == nil {
		config = DefaultConnectionConfig()
	}

	// CRITICAL: Disable Nagle's algorithm to guarantee packet fragmentation
	// This ensures each Write() call creates a separate TCP segment
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
	}

	return &ModifiedConnectionV2{
		Conn:       conn,
		Modifier:   modifier,
		Request:    request,
		Config:     config,
		writeStats: NewWriteStats(),
	}
}

// Read implements net.Conn Read method with enhanced metrics
func (mc *ModifiedConnectionV2) Read(b []byte) (n int, err error) {
	start := time.Now()
	n, err = mc.Conn.Read(b)

	// Record read metrics
	mc.writeStats.mu.Lock()
	mc.writeStats.LastWriteTime = time.Now()
	mc.writeStats.TotalBytes += uint64(n)
	mc.writeStats.mu.Unlock()

	if err != nil {
		return n, err
	}

	// Process inbound data
	modified := mc.Modifier.Process(b[:n], DirectionInbound)
	copy(b, modified)

	// Update latency metrics
	latency := time.Since(start)
	mc.writeStats.mu.Lock()
	if mc.writeStats.TotalWrites > 0 {
		mc.writeStats.AverageLatency = (mc.writeStats.AverageLatency*9 + latency) / 10 // 90% retention
	} else {
		mc.writeStats.AverageLatency = latency
	}
	mc.writeStats.mu.Unlock()

	return len(modified), nil
}

// Write implements net.Conn Write method with real fragmentation support
func (mc *ModifiedConnectionV2) Write(b []byte) (n int, err error) {
	start := time.Now()

	// Choose write method based on modifier capabilities
	if mc.Modifier.SupportsFragmentation() {
		n, err = mc.writeFragmented(b)
	} else {
		n, err = mc.writeLegacy(b)
	}

	// Update write statistics
	mc.writeStats.mu.Lock()
	mc.writeStats.TotalWrites++
	mc.writeStats.TotalBytes += uint64(n)
	mc.writeStats.LastWriteTime = time.Now()

	if err != nil {
		mc.writeStats.ErrorCount++
	}

	latency := time.Since(start)
	if mc.writeStats.TotalWrites > 0 {
		mc.writeStats.AverageLatency = (mc.writeStats.AverageLatency*9 + latency) / 10
	} else {
		mc.writeStats.AverageLatency = latency
	}
	mc.writeStats.mu.Unlock()

	return n, err
}

// writeFragmented performs real network-level fragmentation
func (mc *ModifiedConnectionV2) writeFragmented(b []byte) (int, error) {
	// Get fragmentation configuration
	var antiNagleDelay time.Duration
	if fragModifier, ok := mc.Modifier.(FragmentingModifier); ok {
		config := fragModifier.GetFragmentationConfig()
		antiNagleDelay = config.AntiNagleDelay
	} else {
		antiNagleDelay = mc.Config.AntiNagleDelay
	}

	// Process data into chunks
	chunks := mc.Modifier.ProcessToChunks(b, DirectionOutbound)

	mc.writeStats.mu.Lock()
	mc.writeStats.FragmentedWrites++
	mc.writeStats.mu.Unlock()

	// Write each chunk separately to ensure network-level fragmentation
	totalWritten := 0
	for i, chunk := range chunks {
		// Write chunk with timeout
		if mc.Config.WriteTimeout > 0 {
			mc.Conn.SetWriteDeadline(time.Now().Add(mc.Config.WriteTimeout))
		}

		written, err := mc.writeChunkWithRetry(chunk)
		if err != nil {
			return totalWritten, err
		}

		totalWritten += written

		// Apply anti-Nagle delay between fragments (except for last chunk)
		if i < len(chunks)-1 && antiNagleDelay > 0 {
			time.Sleep(antiNagleDelay)
		}
	}

	return totalWritten, nil
}

// writeLegacy handles non-fragmenting modifiers for backward compatibility
func (mc *ModifiedConnectionV2) writeLegacy(b []byte) (int, error) {
	// Process data through legacy modifier
	modified := mc.Modifier.Process(b, DirectionOutbound)

	// Write processed data in a single call
	return mc.writeChunkWithRetry(modified)
}

// writeChunkWithRetry writes a chunk with retry logic
func (mc *ModifiedConnectionV2) writeChunkWithRetry(chunk []byte) (int, error) {
	var lastErr error
	maxRetries := mc.Config.MaxWriteRetries

	for attempt := 0; attempt <= maxRetries; attempt++ {
		written, err := mc.Conn.Write(chunk)
		if err == nil {
			return written, nil
		}

		lastErr = err

		// Retry on temporary errors
		if isTemporaryError(err) && attempt < maxRetries {
			// Exponential backoff with jitter
			backoff := time.Duration(1<<uint(attempt)) * time.Millisecond
			if backoff > 100*time.Millisecond {
				backoff = 100 * time.Millisecond
			}
			time.Sleep(backoff)
			continue
		}

		// Break on permanent errors or max retries exceeded
		break
	}

	return 0, lastErr
}

// GetWriteStats returns current write statistics
func (mc *ModifiedConnectionV2) GetWriteStats() *WriteStats {
	mc.writeStats.mu.RLock()
	defer mc.writeStats.mu.RUnlock()

	// Return a copy to prevent external modification
	statsCopy := *mc.writeStats
	return &statsCopy
}

// ResetStats resets write statistics
func (mc *ModifiedConnectionV2) ResetStats() {
	mc.writeStats.mu.Lock()
	defer mc.writeStats.mu.Unlock()

	mc.writeStats = NewWriteStats()
}

// SetConfig updates connection configuration
func (mc *ModifiedConnectionV2) SetConfig(config *ConnectionConfig) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.Config = config
}

// GetModifier returns the current modifier
func (mc *ModifiedConnectionV2) GetModifier() ModifierV2 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return mc.Modifier
}

// SetModifier updates the modifier (useful for dynamic modifier switching)
func (mc *ModifiedConnectionV2) SetModifier(modifier ModifierV2) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.Modifier = modifier
}

// NewWriteStats creates a new write statistics tracker
func NewWriteStats() *WriteStats {
	return &WriteStats{
		LastWriteTime: time.Now(),
	}
}

// isTemporaryError determines if an error is temporary and worth retrying
func isTemporaryError(err error) bool {
	if err == nil {
		return false
	}

	// Common temporary network errors
	if netErr, ok := err.(net.Error); ok {
		return netErr.Temporary() || netErr.Timeout()
	}

	// Add more specific error patterns as needed
	errorStr := err.Error()
	temporaryErrors := []string{
		"connection reset by peer",
		"broken pipe",
		"network is unreachable",
		"no route to host",
	}

	for _, tempErr := range temporaryErrors {
		if contains(errorStr, tempErr) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsMiddle(s, substr))))
}

// containsMiddle checks for substring in the middle of a string
func containsMiddle(s, substr string) bool {
	for i := 1; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

package performance

import (
	"net"
	"sync"
	"time"
)

// ConnectionPool manages a pool of reusable connections
type ConnectionPool struct {
	connections        chan *PooledConnection
	maxConnections     int
	currentConnections int
	factory            ConnectionFactory
	mu                 sync.RWMutex
	stats              *PoolStats
	config             *PoolConfig
}

// PooledConnection represents a connection in the pool
type PooledConnection struct {
	conn         net.Conn
	createdAt    time.Time
	lastUsed     time.Time
	usageCount   int
	isActive     bool
	pool         *ConnectionPool
	mu           sync.Mutex
}

// ConnectionFactory creates new connections
type ConnectionFactory interface {
	CreateConnection() (net.Conn, error)
	ValidateConnection(conn net.Conn) bool
	CloseConnection(conn net.Conn) error
}

// PoolStats tracks pool statistics
type PoolStats struct {
	Created       int64
	Reused        int64
	Destroyed      int64
	Errors         int64
	TotalRequests  int64
	HitRate        float64
	AverageLatency time.Duration
	mu             sync.RWMutex
}

// PoolConfig defines pool configuration
type PoolConfig struct {
	MaxConnections     int           `yaml:"max_connections"`
	MaxIdleTime        time.Duration `yaml:"max_idle_time"`
	MaxLifetime        time.Duration `yaml:"max_lifetime"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
	EnableMetrics      bool          `yaml:"enable_metrics"`
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(factory ConnectionFactory, maxConnections int, config *PoolConfig) *ConnectionPool {
	if config == nil {
		config = &PoolConfig{
			MaxConnections:     maxConnections,
			MaxIdleTime:        30 * time.Second,
			MaxLifetime:        5 * time.Minute,
			HealthCheckInterval: 10 * time.Second,
			EnableMetrics:      true,
		}
	}

	pool := &ConnectionPool{
		connections:    make(chan *PooledConnection, maxConnections),
		maxConnections: maxConnections,
		factory:       factory,
		stats:          &PoolStats{},
		config:         config,
	}

	// Start health checker
	go pool.healthChecker()

	return pool
}

// Get retrieves a connection from the pool or creates a new one
func (cp *ConnectionPool) Get() (*PooledConnection, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.stats.mu.Lock()
	cp.stats.TotalRequests++
	cp.stats.mu.Unlock()

	// Try to get from pool first
	select {
	case pooledConn := <-cp.connections:
		if pooledConn.isValid() {
			cp.stats.mu.Lock()
			cp.stats.Reused++
			cp.updateHitRate()
			cp.stats.mu.Unlock()
			
			pooledConn.mu.Lock()
			pooledConn.lastUsed = time.Now()
			pooledConn.usageCount++
			pooledConn.mu.Unlock()
			
			return pooledConn, nil
		} else {
			// Invalid connection, destroy it
			cp.destroyConnection(pooledConn)
		}
	default:
		// No connections available in pool
	}

	// Create new connection if we haven't reached max
	if cp.currentConnections < cp.maxConnections {
		conn, err := cp.factory.CreateConnection()
		if err != nil {
			cp.stats.mu.Lock()
			cp.stats.Errors++
			cp.stats.mu.Unlock()
			return nil, err
		}

		pooledConn := &PooledConnection{
			conn:      conn,
			createdAt: time.Now(),
			lastUsed:  time.Now(),
			isActive:  true,
			pool:      cp,
		}

		cp.currentConnections++
		cp.stats.mu.Lock()
		cp.stats.Created++
		cp.updateHitRate()
		cp.stats.mu.Unlock()

		return pooledConn, nil
	}

	return nil, ErrPoolExhausted
}

// Put returns a connection to the pool
func (cp *ConnectionPool) Put(pooledConn *PooledConnection) {
	if pooledConn == nil || !pooledConn.isActive {
		return
	}

	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Check if connection is still valid
	if !pooledConn.isValid() {
		cp.destroyConnection(pooledConn)
		return
	}

	// Try to return to pool
	select {
	case cp.connections <- pooledConn:
		pooledConn.mu.Lock()
		pooledConn.lastUsed = time.Now()
		pooledConn.mu.Unlock()
	default:
		// Pool is full, destroy the connection
		cp.destroyConnection(pooledConn)
	}
}

// Close closes the connection pool and all connections
func (cp *ConnectionPool) Close() error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	close(cp.connections)
	
	// Close all connections in the pool
	for pooledConn := range cp.connections {
		cp.destroyConnection(pooledConn)
	}

	return nil
}

// isValid checks if a connection is still valid
func (pc *PooledConnection) isValid() bool {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if !pc.isActive || pc.conn == nil {
		return false
	}

	// Check age
	if time.Since(pc.createdAt) > pc.pool.config.MaxLifetime {
		return false
	}

	// Check idle time
	if time.Since(pc.lastUsed) > pc.pool.config.MaxIdleTime {
		return false
	}

	// Validate connection using factory
	return pc.pool.factory.ValidateConnection(pc.conn)
}

// destroyConnection destroys a pooled connection
func (cp *ConnectionPool) destroyConnection(pooledConn *PooledConnection) {
	if pooledConn == nil {
		return
	}

	pooledConn.mu.Lock()
	defer pooledConn.mu.Unlock()

	if pooledConn.isActive {
		pooledConn.isActive = false
		cp.currentConnections--
		
		if err := cp.factory.CloseConnection(pooledConn.conn); err != nil {
			cp.stats.mu.Lock()
			cp.stats.Errors++
			cp.stats.mu.Unlock()
		}
		
		cp.stats.mu.Lock()
		cp.stats.Destroyed++
		cp.stats.mu.Unlock()
	}
}

// updateHitRate updates the pool hit rate
func (cp *ConnectionPool) updateHitRate() {
	if cp.stats.TotalRequests > 0 {
		cp.stats.HitRate = float64(cp.stats.Reused) / float64(cp.stats.TotalRequests)
	}
}

// healthChecker periodically checks connection health
func (cp *ConnectionPool) healthChecker() {
	ticker := time.NewTicker(cp.config.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		cp.performHealthCheck()
	}
}

// performHealthCheck checks the health of pooled connections
func (cp *ConnectionPool) performHealthCheck() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Check a sample of connections
	sampleSize := len(cp.connections)
	if sampleSize > 10 {
		sampleSize = 10
	}

	tempConnections := make([]*PooledConnection, 0, sampleSize)
	
	// Extract sample connections
	for i := 0; i < sampleSize && len(cp.connections) > 0; i++ {
		select {
		case conn := <-cp.connections:
			if conn.isValid() {
				tempConnections = append(tempConnections, conn)
			} else {
				cp.destroyConnection(conn)
			}
		default:
			break
		}
	}

	// Return valid connections to pool
	for _, conn := range tempConnections {
		select {
		case cp.connections <- conn:
		default:
			cp.destroyConnection(conn)
		}
	}
}

// GetStats returns pool statistics
func (cp *ConnectionPool) GetStats() *PoolStats {
	cp.stats.mu.RLock()
	defer cp.stats.mu.RUnlock()

	// Return a copy to avoid race conditions
	return &PoolStats{
		Created:       cp.stats.Created,
		Reused:        cp.stats.Reused,
		Destroyed:      cp.stats.Destroyed,
		Errors:         cp.stats.Errors,
		TotalRequests:  cp.stats.TotalRequests,
		HitRate:        cp.stats.HitRate,
		AverageLatency: cp.stats.AverageLatency,
	}
}

// GetAvailableConnections returns the number of available connections
func (cp *ConnectionPool) GetAvailableConnections() int {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return len(cp.connections)
}

// GetActiveConnections returns the number of active connections
func (cp *ConnectionPool) GetActiveConnections() int {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return cp.currentConnections
}

// ResetStats resets pool statistics
func (cp *ConnectionPool) ResetStats() {
	cp.stats.mu.Lock()
	defer cp.stats.mu.Unlock()

	cp.stats.Created = 0
	cp.stats.Reused = 0
	cp.stats.Destroyed = 0
	cp.stats.Errors = 0
	cp.stats.TotalRequests = 0
	cp.stats.HitRate = 0
	cp.stats.AverageLatency = 0
}

// Write writes data to the underlying connection
func (pc *PooledConnection) Write(b []byte) (int, error) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if !pc.isActive || pc.conn == nil {
		return 0, ErrConnectionClosed
	}

	start := time.Now()
	n, err := pc.conn.Write(b)
	latency := time.Since(start)

	// Update average latency
	pc.pool.stats.mu.Lock()
	if pc.pool.stats.AverageLatency == 0 {
		pc.pool.stats.AverageLatency = latency
	} else {
		pc.pool.stats.AverageLatency = (pc.pool.stats.AverageLatency + latency) / 2
	}
	pc.pool.stats.mu.Unlock()

	return n, err
}

// Read reads data from the underlying connection
func (pc *PooledConnection) Read(b []byte) (int, error) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if !pc.isActive || pc.conn == nil {
		return 0, ErrConnectionClosed
	}

	start := time.Now()
	n, err := pc.conn.Read(b)
	latency := time.Since(start)

	// Update average latency
	pc.pool.stats.mu.Lock()
	if pc.pool.stats.AverageLatency == 0 {
		pc.pool.stats.AverageLatency = latency
	} else {
		pc.pool.stats.AverageLatency = (pc.pool.stats.AverageLatency + latency) / 2
	}
	pc.pool.stats.mu.Unlock()

	return n, err
}

// Close returns the connection to the pool instead of actually closing it
func (pc *PooledConnection) Close() error {
	if pc.pool != nil {
		pc.pool.Put(pc)
	}
	return nil
}

// LocalAddr returns the local network address
func (pc *PooledConnection) LocalAddr() net.Addr {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.conn == nil {
		return nil
	}
	return pc.conn.LocalAddr()
}

// RemoteAddr returns the remote network address
func (pc *PooledConnection) RemoteAddr() net.Addr {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.conn == nil {
		return nil
	}
	return pc.conn.RemoteAddr()
}

// SetDeadline sets the read and write deadlines
func (pc *PooledConnection) SetDeadline(t time.Time) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if !pc.isActive || pc.conn == nil {
		return ErrConnectionClosed
	}
	return pc.conn.SetDeadline(t)
}

// SetReadDeadline sets the read deadline
func (pc *PooledConnection) SetReadDeadline(t time.Time) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if !pc.isActive || pc.conn == nil {
		return ErrConnectionClosed
	}
	return pc.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline
func (pc *PooledConnection) SetWriteDeadline(t time.Time) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if !pc.isActive || pc.conn == nil {
		return ErrConnectionClosed
	}
	return pc.conn.SetWriteDeadline(t)
}

// GetUsageStats returns usage statistics for this connection
func (pc *PooledConnection) GetUsageStats() map[string]interface{} {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	return map[string]interface{}{
		"created_at":  pc.createdAt,
		"last_used":   pc.lastUsed,
		"usage_count": pc.usageCount,
		"is_active":   pc.isActive,
		"age":         time.Since(pc.createdAt),
		"idle_time":   time.Since(pc.lastUsed),
	}
}

// Errors
var (
	ErrPoolExhausted      = &PoolError{Code: "POOL_EXHAUSTED", Message: "connection pool exhausted"}
	ErrConnectionClosed   = &PoolError{Code: "CONN_CLOSED", Message: "connection is closed"}
	ErrInvalidConnection  = &PoolError{Code: "INVALID_CONN", Message: "connection is invalid"}
)

// PoolError represents a connection pool error
type PoolError struct {
	Code    string
	Message string
}

func (e *PoolError) Error() string {
	return e.Message + " (" + e.Code + ")"
}

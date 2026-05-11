package performance

import (
	"math/rand"
	"sync"
	"time"
)

// LoadBalancer manages connection distribution across multiple endpoints
type LoadBalancer struct {
	strategies      map[string]LoadBalancingStrategy
	currentStrategy string
	endpoints       []*Endpoint
	healthChecker   *HealthChecker
	mu              sync.RWMutex
	stats           *BalancerStats
	config          *BalancerConfig
}

// Endpoint represents a target endpoint
type Endpoint struct {
	ID            string
	Address       string
	Weight        int
	Healthy       bool
	ResponseTime  time.Duration
	LastCheck     time.Time
	FailureCount  int
	TotalRequests int64
	ActiveConns   int
	MaxConns      int
	mu            sync.RWMutex
}

// LoadBalancingStrategy interface for different load balancing algorithms
type LoadBalancingStrategy interface {
	SelectEndpoint(endpoints []*Endpoint) (*Endpoint, error)
	Name() string
}

// BalancerStats tracks load balancer statistics
type BalancerStats struct {
	TotalRequests       int64
	SuccessfulRequests  int64
	FailedRequests      int64
	AverageResponseTime time.Duration
	EndpointStats       map[string]*EndpointStats
	mu                  sync.RWMutex
}

// EndpointStats tracks individual endpoint statistics
type EndpointStats struct {
	ID                  string
	Requests            int64
	Successes           int64
	Failures            int64
	AverageResponseTime time.Duration
	LastUsed            time.Time
}

// BalancerConfig defines load balancer configuration
type BalancerConfig struct {
	Strategy             string        `yaml:"strategy"`
	HealthCheckInterval  time.Duration `yaml:"health_check_interval"`
	UnhealthyThreshold   int           `yaml:"unhealthy_threshold"`
	HealthyThreshold     int           `yaml:"healthy_threshold"`
	EnableStickySessions bool          `yaml:"enable_sticky_sessions"`
	SessionTimeout       time.Duration `yaml:"session_timeout"`
	RetryAttempts        int           `yaml:"retry_attempts"`
}

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(endpoints []string, config *BalancerConfig) *LoadBalancer {
	if config == nil {
		config = &BalancerConfig{
			Strategy:             "round_robin",
			HealthCheckInterval:  30 * time.Second,
			UnhealthyThreshold:   3,
			HealthyThreshold:     2,
			EnableStickySessions: false,
			SessionTimeout:       5 * time.Minute,
			RetryAttempts:        3,
		}
	}

	lb := &LoadBalancer{
		strategies:      make(map[string]LoadBalancingStrategy),
		currentStrategy: config.Strategy,
		endpoints:       make([]*Endpoint, 0),
		healthChecker:   NewHealthChecker(config),
		stats:           &BalancerStats{EndpointStats: make(map[string]*EndpointStats)},
		config:          config,
	}

	// Initialize endpoints
	for _, addr := range endpoints {
		endpoint := &Endpoint{
			ID:       addr,
			Address:  addr,
			Weight:   1,
			Healthy:  true,
			MaxConns: 100,
		}
		lb.endpoints = append(lb.endpoints, endpoint)
		lb.stats.EndpointStats[addr] = &EndpointStats{ID: addr}
	}

	// Register strategies
	lb.registerStrategies()

	// Start health checker
	go lb.healthChecker.Start(lb.endpoints)

	return lb
}

// registerStrategies registers all available load balancing strategies
func (lb *LoadBalancer) registerStrategies() {
	lb.strategies["round_robin"] = &RoundRobinStrategy{}
	lb.strategies["weighted_round_robin"] = &WeightedRoundRobinStrategy{}
	lb.strategies["least_connections"] = &LeastConnectionsStrategy{}
	lb.strategies["response_time"] = &ResponseTimeStrategy{}
	lb.strategies["random"] = &RandomStrategy{}
	lb.strategies["hash"] = &HashStrategy{}
}

// GetEndpoint selects an endpoint based on the current strategy
func (lb *LoadBalancer) GetEndpoint(key string) (*Endpoint, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	strategy, exists := lb.strategies[lb.currentStrategy]
	if !exists {
		return nil, ErrStrategyNotFound
	}

	// Filter healthy endpoints
	healthyEndpoints := lb.getHealthyEndpoints()
	if len(healthyEndpoints) == 0 {
		return nil, ErrNoHealthyEndpoints
	}

	endpoint, err := strategy.SelectEndpoint(healthyEndpoints)
	if err != nil {
		return nil, err
	}

	// Update statistics
	lb.updateStats(endpoint, true)

	return endpoint, nil
}

// getHealthyEndpoints returns list of healthy endpoints
func (lb *LoadBalancer) getHealthyEndpoints() []*Endpoint {
	healthy := make([]*Endpoint, 0)
	for _, endpoint := range lb.endpoints {
		endpoint.mu.RLock()
		if endpoint.Healthy && endpoint.ActiveConns < endpoint.MaxConns {
			healthy = append(healthy, endpoint)
		}
		endpoint.mu.RUnlock()
	}
	return healthy
}

// updateStats updates load balancer statistics
func (lb *LoadBalancer) updateStats(endpoint *Endpoint, success bool) {
	lb.stats.mu.Lock()
	defer lb.stats.mu.Unlock()

	lb.stats.TotalRequests++
	if success {
		lb.stats.SuccessfulRequests++
	} else {
		lb.stats.FailedRequests++
	}

	// Update endpoint stats
	if endpointStats, exists := lb.stats.EndpointStats[endpoint.ID]; exists {
		endpointStats.Requests++
		if success {
			endpointStats.Successes++
		} else {
			endpointStats.Failures++
		}
		endpointStats.LastUsed = time.Now()
	}

	// Update endpoint counters
	endpoint.mu.Lock()
	endpoint.TotalRequests++
	endpoint.mu.Unlock()
}

// SetStrategy changes the load balancing strategy
func (lb *LoadBalancer) SetStrategy(strategy string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if _, exists := lb.strategies[strategy]; !exists {
		return ErrStrategyNotFound
	}

	lb.currentStrategy = strategy
	return nil
}

// AddEndpoint adds a new endpoint to the load balancer
func (lb *LoadBalancer) AddEndpoint(address string, weight int) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	endpoint := &Endpoint{
		ID:       address,
		Address:  address,
		Weight:   weight,
		Healthy:  true,
		MaxConns: 100,
	}

	lb.endpoints = append(lb.endpoints, endpoint)
	lb.stats.EndpointStats[address] = &EndpointStats{ID: address}

	return nil
}

// RemoveEndpoint removes an endpoint from the load balancer
func (lb *LoadBalancer) RemoveEndpoint(address string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i, endpoint := range lb.endpoints {
		if endpoint.ID == address {
			lb.endpoints = append(lb.endpoints[:i], lb.endpoints[i+1:]...)
			delete(lb.stats.EndpointStats, address)
			return nil
		}
	}

	return ErrEndpointNotFound
}

// GetStats returns load balancer statistics
func (lb *LoadBalancer) GetStats() *BalancerStats {
	lb.stats.mu.RLock()
	defer lb.stats.mu.RUnlock()

	// Return a copy to avoid race conditions
	statsCopy := &BalancerStats{
		TotalRequests:       lb.stats.TotalRequests,
		SuccessfulRequests:  lb.stats.SuccessfulRequests,
		FailedRequests:      lb.stats.FailedRequests,
		AverageResponseTime: lb.stats.AverageResponseTime,
		EndpointStats:       make(map[string]*EndpointStats),
	}

	for k, v := range lb.stats.EndpointStats {
		statsCopy.EndpointStats[k] = &EndpointStats{
			ID:                  v.ID,
			Requests:            v.Requests,
			Successes:           v.Successes,
			Failures:            v.Failures,
			AverageResponseTime: v.AverageResponseTime,
			LastUsed:            v.LastUsed,
		}
	}

	return statsCopy
}

// GetEndpoints returns all endpoints with their current status
func (lb *LoadBalancer) GetEndpoints() []*Endpoint {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	endpoints := make([]*Endpoint, len(lb.endpoints))
	copy(endpoints, lb.endpoints)
	return endpoints
}

// Close closes the load balancer
func (lb *LoadBalancer) Close() error {
	if lb.healthChecker != nil {
		lb.healthChecker.Stop()
	}
	return nil
}

// RoundRobinStrategy implements round-robin load balancing
type RoundRobinStrategy struct {
	current int
	mu      sync.Mutex
}

func (rr *RoundRobinStrategy) SelectEndpoint(endpoints []*Endpoint) (*Endpoint, error) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if len(endpoints) == 0 {
		return nil, ErrNoEndpoints
	}

	endpoint := endpoints[rr.current%len(endpoints)]
	rr.current++
	return endpoint, nil
}

func (rr *RoundRobinStrategy) Name() string {
	return "round_robin"
}

// WeightedRoundRobinStrategy implements weighted round-robin load balancing
type WeightedRoundRobinStrategy struct {
	currentWeight int
	mu            sync.Mutex
}

func (wrr *WeightedRoundRobinStrategy) SelectEndpoint(endpoints []*Endpoint) (*Endpoint, error) {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	if len(endpoints) == 0 {
		return nil, ErrNoEndpoints
	}

	// Calculate total weight
	totalWeight := 0
	for _, endpoint := range endpoints {
		totalWeight += endpoint.Weight
	}

	if totalWeight == 0 {
		// Fallback to simple round-robin
		return endpoints[wrr.currentWeight%len(endpoints)], nil
	}

	// Select endpoint based on weight
	wrr.currentWeight = (wrr.currentWeight + 1) % totalWeight
	currentWeight := 0

	for _, endpoint := range endpoints {
		currentWeight += endpoint.Weight
		if wrr.currentWeight < currentWeight {
			return endpoint, nil
		}
	}

	return endpoints[0], nil
}

func (wrr *WeightedRoundRobinStrategy) Name() string {
	return "weighted_round_robin"
}

// LeastConnectionsStrategy implements least connections load balancing
type LeastConnectionsStrategy struct{}

func (lc *LeastConnectionsStrategy) SelectEndpoint(endpoints []*Endpoint) (*Endpoint, error) {
	if len(endpoints) == 0 {
		return nil, ErrNoEndpoints
	}

	var selected *Endpoint
	minConns := int(^uint(0) >> 1) // Max int

	for _, endpoint := range endpoints {
		endpoint.mu.RLock()
		conns := endpoint.ActiveConns
		endpoint.mu.RUnlock()

		if conns < minConns {
			minConns = conns
			selected = endpoint
		}
	}

	if selected == nil {
		selected = endpoints[0]
	}

	return selected, nil
}

func (lc *LeastConnectionsStrategy) Name() string {
	return "least_connections"
}

// ResponseTimeStrategy implements response time based load balancing
type ResponseTimeStrategy struct{}

func (rt *ResponseTimeStrategy) SelectEndpoint(endpoints []*Endpoint) (*Endpoint, error) {
	if len(endpoints) == 0 {
		return nil, ErrNoEndpoints
	}

	var selected *Endpoint
	minResponseTime := time.Hour // Large initial value

	for _, endpoint := range endpoints {
		endpoint.mu.RLock()
		responseTime := endpoint.ResponseTime
		endpoint.mu.RUnlock()

		if responseTime < minResponseTime {
			minResponseTime = responseTime
			selected = endpoint
		}
	}

	if selected == nil {
		selected = endpoints[0]
	}

	return selected, nil
}

func (rt *ResponseTimeStrategy) Name() string {
	return "response_time"
}

// RandomStrategy implements random load balancing
type RandomStrategy struct{}

func (r *RandomStrategy) SelectEndpoint(endpoints []*Endpoint) (*Endpoint, error) {
	if len(endpoints) == 0 {
		return nil, ErrNoEndpoints
	}

	return endpoints[rand.Intn(len(endpoints))], nil
}

func (r *RandomStrategy) Name() string {
	return "random"
}

// HashStrategy implements hash-based load balancing
type HashStrategy struct{}

func (h *HashStrategy) SelectEndpoint(endpoints []*Endpoint) (*Endpoint, error) {
	if len(endpoints) == 0 {
		return nil, ErrNoEndpoints
	}

	// Simple hash based on time (in real implementation, use client IP or session key)
	hash := time.Now().UnixNano()
	index := int(hash % int64(len(endpoints)))
	return endpoints[index], nil
}

func (h *HashStrategy) Name() string {
	return "hash"
}

// HealthChecker monitors endpoint health
type HealthChecker struct {
	interval time.Duration
	stopCh   chan struct{}
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(config *BalancerConfig) *HealthChecker {
	return &HealthChecker{
		interval: config.HealthCheckInterval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins health checking
func (hc *HealthChecker) Start(endpoints []*Endpoint) {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.checkEndpoints(endpoints)
		case <-hc.stopCh:
			return
		}
	}
}

// Stop stops health checking
func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
}

// checkEndpoints checks the health of all endpoints
func (hc *HealthChecker) checkEndpoints(endpoints []*Endpoint) {
	for _, endpoint := range endpoints {
		go hc.checkEndpoint(endpoint)
	}
}

// checkEndpoint checks the health of a single endpoint
func (hc *HealthChecker) checkEndpoint(endpoint *Endpoint) {
	start := time.Now()

	// Simple health check (in real implementation, make actual request)
	healthy := true // Placeholder

	responseTime := time.Since(start)

	endpoint.mu.Lock()
	endpoint.Healthy = healthy
	endpoint.ResponseTime = responseTime
	endpoint.LastCheck = time.Now()

	if !healthy {
		endpoint.FailureCount++
	} else {
		endpoint.FailureCount = 0
	}
	endpoint.mu.Unlock()
}

// Errors
var (
	ErrStrategyNotFound   = &BalancerError{Code: "STRATEGY_NOT_FOUND", Message: "load balancing strategy not found"}
	ErrNoHealthyEndpoints = &BalancerError{Code: "NO_HEALTHY_ENDPOINTS", Message: "no healthy endpoints available"}
	ErrNoEndpoints        = &BalancerError{Code: "NO_ENDPOINTS", Message: "no endpoints available"}
	ErrEndpointNotFound   = &BalancerError{Code: "ENDPOINT_NOT_FOUND", Message: "endpoint not found"}
)

// BalancerError represents a load balancer error
type BalancerError struct {
	Code    string
	Message string
}

func (e *BalancerError) Error() string {
	return e.Message + " (" + e.Code + ")"
}

package distributed

import (
	"context"
	"net"
	"sync"
	"time"
)

// NodeManager manages distributed nodes in the system
type NodeManager struct {
	nodes         map[string]*Node
	localNode     *Node
	coordinator   *Coordinator
	healthChecker *NodeHealthChecker
	loadBalancer  *DistributedLoadBalancer
	config        *NodeManagerConfig
	mu            sync.RWMutex
	running       bool
	stopCh        chan struct{}
}

// Node represents a distributed system node
type Node struct {
	ID            string                 `json:"id"`
	Address       string                 `json:"address"`
	Port          int                    `json:"port"`
	Status        NodeStatus             `json:"status"`
	Capabilities  *NodeCapabilities      `json:"capabilities"`
	Resources     *NodeResources         `json:"resources"`
	LastHeartbeat time.Time              `json:"last_heartbeat"`
	Metadata      map[string]interface{} `json:"metadata"`
	Connection    net.Conn               `json:"-"`
	mu            sync.RWMutex
}

// NodeStatus represents the status of a node
type NodeStatus int

const (
	NodeStatusUnknown NodeStatus = iota
	NodeStatusActive
	NodeStatusInactive
	NodeStatusDegraded
	NodeStatusMaintenance
	NodeStatusFailed
)

// NodeCapabilities represents node capabilities
type NodeCapabilities struct {
	SupportedProtocols   []string `json:"supported_protocols"`
	MaxConnections       int      `json:"max_connections"`
	HasTLS               bool     `json:"has_tls"`
	HasQUIC              bool     `json:"has_quic"`
	HasBehavioralEvasion bool     `json:"has_behavioral_evasion"`
	Version              string   `json:"version"`
}

// NodeResources represents node resource usage
type NodeResources struct {
	CPUUsage          float64 `json:"cpu_usage"`
	MemoryUsage       float64 `json:"memory_usage"`
	NetworkUsage      float64 `json:"network_usage"`
	DiskUsage         float64 `json:"disk_usage"`
	ActiveConnections int     `json:"active_connections"`
	LoadAverage       float64 `json:"load_average"`
}

// Coordinator manages node coordination
type Coordinator struct {
	leader         string
	electionTimer  *time.Timer
	heartbeatTimer *time.Ticker
	mu             sync.RWMutex
}

// NodeHealthChecker monitors node health
type NodeHealthChecker struct {
	checkInterval    time.Duration
	timeout          time.Duration
	failureThreshold int
	mu               sync.RWMutex
}

// DistributedLoadBalancer balances load across nodes
type DistributedLoadBalancer struct {
	strategy LoadBalancingStrategy
	nodes    map[string]*Node
	mu       sync.RWMutex
}

// LoadBalancingStrategy interface for distributed load balancing
type LoadBalancingStrategy interface {
	SelectNode(nodes []*Node, criteria *SelectionCriteria) (*Node, error)
	Name() string
}

// SelectionCriteria defines node selection criteria
type SelectionCriteria struct {
	RequiredCapabilities []string
	MinResources         *NodeResources
	PreferLocal          bool
	ExcludeFailed        bool
}

// NodeManagerConfig defines node manager configuration
type NodeManagerConfig struct {
	NodeID              string        `yaml:"node_id"`
	ListenAddress       string        `yaml:"listen_address"`
	ListenPort          int           `yaml:"listen_port"`
	HeartbeatInterval   time.Duration `yaml:"heartbeat_interval"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
	ElectionTimeout     time.Duration `yaml:"election_timeout"`
	MaxNodes            int           `yaml:"max_nodes"`
	EnableTLS           bool          `yaml:"enable_tls"`
	TLSConfig           *TLSConfig    `yaml:"tls_config"`
}

// TLSConfig defines TLS configuration
type TLSConfig struct {
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
	CAFile   string `yaml:"ca_file"`
}

// NewNodeManager creates a new node manager
func NewNodeManager(config *NodeManagerConfig) *NodeManager {
	if config == nil {
		config = &NodeManagerConfig{
			NodeID:              "node-1",
			ListenAddress:       "0.0.0.0",
			ListenPort:          8081,
			HeartbeatInterval:   10 * time.Second,
			HealthCheckInterval: 30 * time.Second,
			ElectionTimeout:     30 * time.Second,
			MaxNodes:            10,
			EnableTLS:           false,
		}
	}

	nm := &NodeManager{
		nodes:  make(map[string]*Node),
		config: config,
		stopCh: make(chan struct{}),
		coordinator: &Coordinator{
			heartbeatTimer: time.NewTicker(config.HeartbeatInterval),
		},
		healthChecker: &NodeHealthChecker{
			checkInterval:    config.HealthCheckInterval,
			timeout:          10 * time.Second,
			failureThreshold: 3,
		},
		loadBalancer: &DistributedLoadBalancer{
			strategy: &RoundRobinNodeStrategy{},
			nodes:    make(map[string]*Node),
		},
	}

	// Create local node
	nm.localNode = &Node{
		ID:      config.NodeID,
		Address: config.ListenAddress,
		Port:    config.ListenPort,
		Status:  NodeStatusActive,
		Capabilities: &NodeCapabilities{
			SupportedProtocols:   []string{"vless", "hysteria2", "tuic"},
			MaxConnections:       1000,
			HasTLS:               config.EnableTLS,
			HasQUIC:              true,
			HasBehavioralEvasion: true,
			Version:              "1.0.0",
		},
		Resources: &NodeResources{
			CPUUsage:          0.0,
			MemoryUsage:       0.0,
			NetworkUsage:      0.0,
			DiskUsage:         0.0,
			ActiveConnections: 0,
			LoadAverage:       0.0,
		},
		LastHeartbeat: time.Now(),
		Metadata:      make(map[string]interface{}),
	}

	nm.nodes[config.NodeID] = nm.localNode

	return nm
}

// Start starts the node manager
func (nm *NodeManager) Start(ctx context.Context) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if nm.running {
		return ErrNodeManagerAlreadyRunning
	}

	nm.running = true

	// Start coordinator
	go nm.runCoordinator(ctx)

	// Start health checker
	go nm.runHealthChecker(ctx)

	// Start load balancer
	go nm.runLoadBalancer(ctx)

	return nil
}

// Stop stops the node manager
func (nm *NodeManager) Stop() error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if !nm.running {
		return ErrNodeManagerNotRunning
	}

	nm.running = false
	close(nm.stopCh)

	// Stop coordinator
	if nm.coordinator.heartbeatTimer != nil {
		nm.coordinator.heartbeatTimer.Stop()
	}

	// Close all node connections
	for _, node := range nm.nodes {
		if node.Connection != nil {
			node.Connection.Close()
		}
	}

	return nil
}

// AddNode adds a new node to the cluster
func (nm *NodeManager) AddNode(node *Node) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if len(nm.nodes) >= nm.config.MaxNodes {
		return ErrMaxNodesReached
	}

	if _, exists := nm.nodes[node.ID]; exists {
		return ErrNodeAlreadyExists
	}

	node.Status = NodeStatusActive
	node.LastHeartbeat = time.Now()
	nm.nodes[node.ID] = node

	return nil
}

// RemoveNode removes a node from the cluster
func (nm *NodeManager) RemoveNode(nodeID string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	node, exists := nm.nodes[nodeID]
	if !exists {
		return ErrNodeNotFound
	}

	if node.Connection != nil {
		node.Connection.Close()
	}

	delete(nm.nodes, nodeID)

	return nil
}

// GetNode returns a node by ID
func (nm *NodeManager) GetNode(nodeID string) (*Node, error) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	node, exists := nm.nodes[nodeID]
	if !exists {
		return nil, ErrNodeNotFound
	}

	return node, nil
}

// GetAllNodes returns all nodes in the cluster
func (nm *NodeManager) GetAllNodes() []*Node {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	nodes := make([]*Node, 0, len(nm.nodes))
	for _, node := range nm.nodes {
		nodes = append(nodes, node)
	}

	return nodes
}

// GetActiveNodes returns all active nodes
func (nm *NodeManager) GetActiveNodes() []*Node {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	nodes := make([]*Node, 0)
	for _, node := range nm.nodes {
		if node.Status == NodeStatusActive {
			nodes = append(nodes, node)
		}
	}

	return nodes
}

// SelectNode selects a node based on criteria
func (nm *NodeManager) SelectNode(criteria *SelectionCriteria) (*Node, error) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	activeNodes := nm.GetActiveNodes()
	if len(activeNodes) == 0 {
		return nil, ErrNoActiveNodes
	}

	// Filter nodes based on criteria
	filteredNodes := nm.filterNodes(activeNodes, criteria)
	if len(filteredNodes) == 0 {
		return nil, ErrNoSuitableNodes
	}

	// Use load balancer to select node
	return nm.loadBalancer.strategy.SelectNode(filteredNodes, criteria)
}

// filterNodes filters nodes based on selection criteria
func (nm *NodeManager) filterNodes(nodes []*Node, criteria *SelectionCriteria) []*Node {
	filtered := make([]*Node, 0)

	for _, node := range nodes {
		if nm.matchesCriteria(node, criteria) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// matchesCriteria checks if a node matches the selection criteria
func (nm *NodeManager) matchesCriteria(node *Node, criteria *SelectionCriteria) bool {
	// Check required capabilities
	if len(criteria.RequiredCapabilities) > 0 {
		nodeCaps := node.Capabilities
		for _, requiredCap := range criteria.RequiredCapabilities {
			found := false
			for _, nodeCap := range nodeCaps.SupportedProtocols {
				if nodeCap == requiredCap {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// Check minimum resources
	if criteria.MinResources != nil {
		resources := node.Resources
		if criteria.MinResources.CPUUsage > 0 && resources.CPUUsage > criteria.MinResources.CPUUsage {
			return false
		}
		if criteria.MinResources.MemoryUsage > 0 && resources.MemoryUsage > criteria.MinResources.MemoryUsage {
			return false
		}
	}

	// Exclude failed nodes
	if criteria.ExcludeFailed && node.Status == NodeStatusFailed {
		return false
	}

	return true
}

// UpdateNodeStatus updates the status of a node
func (nm *NodeManager) UpdateNodeStatus(nodeID string, status NodeStatus) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	node, exists := nm.nodes[nodeID]
	if !exists {
		return ErrNodeNotFound
	}

	node.mu.Lock()
	node.Status = status
	node.mu.Unlock()

	return nil
}

// UpdateNodeResources updates node resource usage
func (nm *NodeManager) UpdateNodeResources(nodeID string, resources *NodeResources) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	node, exists := nm.nodes[nodeID]
	if !exists {
		return ErrNodeNotFound
	}

	node.mu.Lock()
	node.Resources = resources
	node.mu.Unlock()

	return nil
}

// runCoordinator runs the coordinator logic
func (nm *NodeManager) runCoordinator(ctx context.Context) {
	ticker := nm.coordinator.heartbeatTimer
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			nm.sendHeartbeat()
		case <-nm.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// sendHeartbeat sends heartbeat to other nodes
func (nm *NodeManager) sendHeartbeat() {
	// Update local node heartbeat
	nm.localNode.mu.Lock()
	nm.localNode.LastHeartbeat = time.Now()
	nm.localNode.mu.Unlock()

	// Send heartbeat to other nodes (simplified)
	for _, node := range nm.nodes {
		if node.ID != nm.localNode.ID {
			go nm.sendHeartbeatToNode(node)
		}
	}
}

// sendHeartbeatToNode sends heartbeat to a specific node
func (nm *NodeManager) sendHeartbeatToNode(node *Node) {
	// Simplified heartbeat implementation
	// In real implementation, this would send actual heartbeat message
}

// runHealthChecker runs the health checker
func (nm *NodeManager) runHealthChecker(ctx context.Context) {
	ticker := time.NewTicker(nm.healthChecker.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			nm.checkNodeHealth()
		case <-nm.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// checkNodeHealth checks the health of all nodes
func (nm *NodeManager) checkNodeHealth() {
	for _, node := range nm.nodes {
		go nm.checkSingleNodeHealth(node)
	}
}

// checkSingleNodeHealth checks the health of a single node
func (nm *NodeManager) checkSingleNodeHealth(node *Node) {
	// Simplified health check
	// In real implementation, this would perform actual health check
	if time.Since(node.LastHeartbeat) > nm.healthChecker.timeout {
		nm.UpdateNodeStatus(node.ID, NodeStatusFailed)
	}
}

// runLoadBalancer runs the load balancer
func (nm *NodeManager) runLoadBalancer(ctx context.Context) {
	// Load balancer logic would run here
	for {
		select {
		case <-nm.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// GetClusterStats returns cluster statistics
func (nm *NodeManager) GetClusterStats() *ClusterStats {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	stats := &ClusterStats{
		TotalNodes:       len(nm.nodes),
		ActiveNodes:      0,
		FailedNodes:      0,
		TotalConnections: 0,
		AverageLoad:      0.0,
	}

	for _, node := range nm.nodes {
		switch node.Status {
		case NodeStatusActive:
			stats.ActiveNodes++
		case NodeStatusFailed:
			stats.FailedNodes++
		}

		stats.TotalConnections += node.Resources.ActiveConnections
		stats.AverageLoad += node.Resources.LoadAverage
	}

	if len(nm.nodes) > 0 {
		stats.AverageLoad /= float64(len(nm.nodes))
	}

	return stats
}

// ClusterStats represents cluster statistics
type ClusterStats struct {
	TotalNodes       int     `json:"total_nodes"`
	ActiveNodes      int     `json:"active_nodes"`
	FailedNodes      int     `json:"failed_nodes"`
	TotalConnections int     `json:"total_connections"`
	AverageLoad      float64 `json:"average_load"`
}

// RoundRobinNodeStrategy implements round-robin node selection
type RoundRobinNodeStrategy struct {
	current int
	mu      sync.Mutex
}

func (rr *RoundRobinNodeStrategy) SelectNode(nodes []*Node, criteria *SelectionCriteria) (*Node, error) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if len(nodes) == 0 {
		return nil, ErrNoNodes
	}

	node := nodes[rr.current%len(nodes)]
	rr.current++
	return node, nil
}

func (rr *RoundRobinNodeStrategy) Name() string {
	return "round_robin"
}

// Errors
var (
	ErrNodeManagerAlreadyRunning = &NodeManagerError{Code: "ALREADY_RUNNING", Message: "node manager is already running"}
	ErrNodeManagerNotRunning     = &NodeManagerError{Code: "NOT_RUNNING", Message: "node manager is not running"}
	ErrMaxNodesReached           = &NodeManagerError{Code: "MAX_NODES", Message: "maximum number of nodes reached"}
	ErrNodeAlreadyExists         = &NodeManagerError{Code: "NODE_EXISTS", Message: "node already exists"}
	ErrNodeNotFound              = &NodeManagerError{Code: "NODE_NOT_FOUND", Message: "node not found"}
	ErrNoActiveNodes             = &NodeManagerError{Code: "NO_ACTIVE_NODES", Message: "no active nodes available"}
	ErrNoSuitableNodes           = &NodeManagerError{Code: "NO_SUITABLE_NODES", Message: "no suitable nodes found"}
	ErrNoNodes                   = &NodeManagerError{Code: "NO_NODES", Message: "no nodes available"}
)

// NodeManagerError represents a node manager error
type NodeManagerError struct {
	Code    string
	Message string
}

func (e *NodeManagerError) Error() string {
	return e.Message + " (" + e.Code + ")"
}

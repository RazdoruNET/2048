package distributed

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// NodeMonitor monitors distributed node performance
type NodeMonitor struct {
	nodes            map[string]*MonitoredNode
	metricsCollector *NodeMetricsCollector
	alertManager     *NodeAlertManager
	dashboard        *NodeDashboard
	config           *NodeMonitoringConfig
	mu               sync.RWMutex
	running          bool
	stopCh           chan struct{}
}

// MonitoredNode represents a node being monitored
type MonitoredNode struct {
	ID             string
	Address        string
	Status         NodeStatus
	Resources      *NodeResources
	Performance    *NodePerformance
	Network        *NodeNetwork
	Applications   []*NodeApplication
	LastUpdate     time.Time
	MetricsHistory []NodeMetricsSnapshot
	Alerts         []NodeAlert
	mu             sync.RWMutex
}

// NodePerformance represents node performance metrics
type NodePerformance struct {
	CPUUsage     float64       `json:"cpu_usage"`
	MemoryUsage  float64       `json:"memory_usage"`
	DiskUsage    float64       `json:"disk_usage"`
	NetworkIO    int64         `json:"network_io"`
	DiskIO       int64         `json:"disk_io"`
	ResponseTime time.Duration `json:"response_time"`
	Throughput   int64         `json:"throughput"`
	ErrorRate    float64       `json:"error_rate"`
}

// NodeNetwork represents node network metrics
type NodeNetwork struct {
	Latency           time.Duration `json:"latency"`
	Bandwidth         int64         `json:"bandwidth"`
	PacketLoss        float64       `json:"packet_loss"`
	ConnectionsActive int           `json:"connections_active"`
	ConnectionsTotal  int           `json:"connections_total"`
}

// NodeApplication represents an application running on node
type NodeApplication struct {
	Name            string    `json:"name"`
	Version         string    `json:"version"`
	Status          string    `json:"status"`
	CPUUsage        float64   `json:"cpu_usage"`
	MemoryUsage     float64   `json:"memory_usage"`
	ConnectionCount int       `json:"connection_count"`
	LastRestart     time.Time `json:"last_restart"`
}

// NodeMetricsSnapshot represents a metrics snapshot
type NodeMetricsSnapshot struct {
	Timestamp   time.Time        `json:"timestamp"`
	Performance *NodePerformance `json:"performance"`
	Network     *NodeNetwork     `json:"network"`
	Resources   *NodeResources   `json:"resources"`
}

// NodeAlert represents a node alert
type NodeAlert struct {
	ID        string                 `json:"id"`
	NodeID    string                 `json:"node_id"`
	Type      AlertType              `json:"type"`
	Severity  AlertSeverity          `json:"severity"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Resolved  bool                   `json:"resolved"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// AlertType represents the type of alert
type AlertType int

const (
	AlertTypeCPU AlertType = iota
	AlertTypeMemory
	AlertTypeDisk
	AlertTypeNetwork
	AlertTypeApplication
	AlertTypeConnection
	AlertTypePerformance
)

// AlertSeverity represents alert severity
type AlertSeverity int

const (
	SeverityInfo AlertSeverity = iota
	SeverityWarning
	SeverityCritical
	SeverityEmergency
)

// NodeMetricsCollector collects metrics from nodes
type NodeMetricsCollector struct {
	collectInterval time.Duration
	metricsBuffer   map[string][]NodeMetricsSnapshot
	mu              sync.RWMutex
}

// NodeAlertManager manages node alerts
type NodeAlertManager struct {
	rules          []*NodeAlertRule
	activeAlerts   map[string][]NodeAlert
	notificationCh chan NodeAlert
	mu             sync.RWMutex
}

// NodeAlertRule represents an alert rule
type NodeAlertRule struct {
	Name      string        `yaml:"name"`
	Metric    string        `yaml:"metric"`
	Condition string        `yaml:"condition"`
	Threshold float64       `yaml:"threshold"`
	Duration  time.Duration `yaml:"duration"`
	Severity  AlertSeverity `yaml:"severity"`
	Enabled   bool          `yaml:"enabled"`
}

// NodeDashboard provides monitoring dashboard
type NodeDashboard struct {
	server    *NodeDashboardServer
	endpoints map[string]func() interface{}
	mu        sync.RWMutex
}

// NodeDashboardServer serves the dashboard
type NodeDashboardServer struct {
	port int
	mu   sync.RWMutex
}

// NodeMonitoringConfig defines node monitoring configuration
type NodeMonitoringConfig struct {
	Enabled           bool          `yaml:"enabled"`
	CollectInterval   time.Duration `yaml:"collect_interval"`
	RetentionPeriod   time.Duration `yaml:"retention_period"`
	AlertingEnabled   bool          `yaml:"alerting_enabled"`
	DashboardEnabled  bool          `yaml:"dashboard_enabled"`
	DashboardPort     int           `yaml:"dashboard_port"`
	MetricsBufferSize int           `yaml:"metrics_buffer_size"`
	EnableAutoScaling bool          `yaml:"enable_auto_scaling"`
	ScalingThreshold  float64       `yaml:"scaling_threshold"`
}

// NewNodeMonitor creates a new node monitor
func NewNodeMonitor(config *NodeMonitoringConfig) *NodeMonitor {
	if config == nil {
		config = &NodeMonitoringConfig{
			Enabled:           true,
			CollectInterval:   30 * time.Second,
			RetentionPeriod:   24 * time.Hour,
			AlertingEnabled:   true,
			DashboardEnabled:  true,
			DashboardPort:     8085,
			MetricsBufferSize: 1000,
			EnableAutoScaling: false,
			ScalingThreshold:  80.0,
		}
	}

	nm := &NodeMonitor{
		nodes: make(map[string]*MonitoredNode),
		metricsCollector: &NodeMetricsCollector{
			collectInterval: config.CollectInterval,
			metricsBuffer:   make(map[string][]NodeMetricsSnapshot),
		},
		alertManager: &NodeAlertManager{
			rules:          []*NodeAlertRule{},
			activeAlerts:   make(map[string][]NodeAlert),
			notificationCh: make(chan NodeAlert, 100),
		},
		dashboard: &NodeDashboard{
			server:    &NodeDashboardServer{port: config.DashboardPort},
			endpoints: make(map[string]func() interface{}),
		},
		config: config,
		stopCh: make(chan struct{}),
	}

	// Initialize default alert rules
	nm.initializeAlertRules()

	return nm
}

// Start starts the node monitor
func (nm *NodeMonitor) Start(ctx context.Context) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if nm.running {
		return ErrNodeMonitorAlreadyRunning
	}

	if !nm.config.Enabled {
		return nil
	}

	nm.running = true

	// Start metrics collection
	go nm.runMetricsCollection(ctx)

	// Start alert processing
	if nm.config.AlertingEnabled {
		go nm.runAlertProcessing(ctx)
	}

	// Start dashboard
	if nm.config.DashboardEnabled {
		go nm.runDashboard(ctx)
	}

	// Start auto-scaling
	if nm.config.EnableAutoScaling {
		go nm.runAutoScaling(ctx)
	}

	return nil
}

// Stop stops the node monitor
func (nm *NodeMonitor) Stop() error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if !nm.running {
		return ErrNodeMonitorNotRunning
	}

	nm.running = false
	close(nm.stopCh)

	return nil
}

// AddNode adds a node for monitoring
func (nm *NodeMonitor) AddNode(node *MonitoredNode) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if _, exists := nm.nodes[node.ID]; exists {
		return ErrMonitorNodeAlreadyExists
	}

	node.LastUpdate = time.Now()
	node.MetricsHistory = make([]NodeMetricsSnapshot, 0)
	node.Alerts = make([]NodeAlert, 0)

	nm.nodes[node.ID] = node

	return nil
}

// RemoveNode removes a node from monitoring
func (nm *NodeMonitor) RemoveNode(nodeID string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if _, exists := nm.nodes[nodeID]; !exists {
		return ErrMonitorNodeNotFound
	}

	delete(nm.nodes, nodeID)
	delete(nm.metricsCollector.metricsBuffer, nodeID)

	return nil
}

// GetNode returns a monitored node
func (nm *NodeMonitor) GetNode(nodeID string) (*MonitoredNode, error) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	node, exists := nm.nodes[nodeID]
	if !exists {
		return nil, ErrMonitorNodeNotFound
	}

	return node, nil
}

// GetAllNodes returns all monitored nodes
func (nm *NodeMonitor) GetAllNodes() []*MonitoredNode {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	nodes := make([]*MonitoredNode, 0, len(nm.nodes))
	for _, node := range nm.nodes {
		nodes = append(nodes, node)
	}

	return nodes
}

// runMetricsCollection runs continuous metrics collection
func (nm *NodeMonitor) runMetricsCollection(ctx context.Context) {
	ticker := time.NewTicker(nm.config.CollectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			nm.collectMetrics()
		case <-nm.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// collectMetrics collects metrics from all nodes
func (nm *NodeMonitor) collectMetrics() {
	for _, node := range nm.nodes {
		go nm.collectNodeMetrics(node)
	}
}

// collectNodeMetrics collects metrics from a specific node
func (nm *NodeMonitor) collectNodeMetrics(node *MonitoredNode) {
	// Simulate metrics collection
	snapshot := NodeMetricsSnapshot{
		Timestamp: time.Now(),
		Performance: &NodePerformance{
			CPUUsage:     float64(time.Now().UnixNano()%100) / 100.0,
			MemoryUsage:  float64(time.Now().UnixNano()%100) / 100.0,
			DiskUsage:    float64(time.Now().UnixNano()%100) / 100.0,
			NetworkIO:    int64(time.Now().UnixNano() % 10000),
			DiskIO:       int64(time.Now().UnixNano() % 10000),
			ResponseTime: time.Duration(10+time.Now().UnixNano()%100) * time.Millisecond,
			Throughput:   int64(time.Now().UnixNano() % 1000),
			ErrorRate:    float64(time.Now().UnixNano()%100) / 100.0,
		},
		Network: &NodeNetwork{
			Latency:           time.Duration(5+time.Now().UnixNano()%50) * time.Millisecond,
			Bandwidth:         int64(time.Now().UnixNano() % 1000000),
			PacketLoss:        float64(time.Now().UnixNano()%10) / 100.0,
			ConnectionsActive: int(time.Now().UnixNano() % 100),
			ConnectionsTotal:  int(time.Now().UnixNano() % 1000),
		},
		Resources: node.Resources,
	}

	// Update node metrics
	node.mu.Lock()
	node.Performance = snapshot.Performance
	node.Network = snapshot.Network
	node.LastUpdate = snapshot.Timestamp
	node.MetricsHistory = append(node.MetricsHistory, snapshot)

	// Maintain buffer size
	if len(node.MetricsHistory) > nm.config.MetricsBufferSize {
		node.MetricsHistory = node.MetricsHistory[1:]
	}
	node.mu.Unlock()

	// Store in collector buffer
	nm.metricsCollector.mu.Lock()
	nm.metricsCollector.metricsBuffer[node.ID] = append(nm.metricsCollector.metricsBuffer[node.ID], snapshot)
	if len(nm.metricsCollector.metricsBuffer[node.ID]) > nm.config.MetricsBufferSize {
		nm.metricsCollector.metricsBuffer[node.ID] = nm.metricsCollector.metricsBuffer[node.ID][1:]
	}
	nm.metricsCollector.mu.Unlock()

	// Check for alerts
	if nm.config.AlertingEnabled {
		nm.checkAlerts(node, snapshot)
	}
}

// checkAlerts checks for alert conditions
func (nm *NodeMonitor) checkAlerts(node *MonitoredNode, snapshot NodeMetricsSnapshot) {
	for _, rule := range nm.alertManager.rules {
		if !rule.Enabled {
			continue
		}

		if nm.evaluateAlertRule(node, rule, snapshot) {
			alert := NodeAlert{
				ID:        generateAlertID(),
				NodeID:    node.ID,
				Type:      nm.getAlertType(rule.Metric),
				Severity:  rule.Severity,
				Message:   nm.formatAlertMessage(rule, snapshot),
				Timestamp: time.Now(),
				Resolved:  false,
			}

			nm.alertManager.notificationCh <- alert
		}
	}
}

// evaluateAlertRule evaluates an alert rule
func (nm *NodeMonitor) evaluateAlertRule(node *MonitoredNode, rule *NodeAlertRule, snapshot NodeMetricsSnapshot) bool {
	var value float64

	switch rule.Metric {
	case "cpu_usage":
		value = snapshot.Performance.CPUUsage
	case "memory_usage":
		value = snapshot.Performance.MemoryUsage
	case "disk_usage":
		value = snapshot.Performance.DiskUsage
	case "error_rate":
		value = snapshot.Performance.ErrorRate
	case "network_latency":
		value = float64(snapshot.Network.Latency.Nanoseconds()) / 1e6 // Convert to ms
	default:
		return false
	}

	return nm.evaluateCondition(value, rule.Condition, rule.Threshold)
}

// evaluateCondition evaluates alert condition
func (nm *NodeMonitor) evaluateCondition(value float64, condition string, threshold float64) bool {
	switch condition {
	case ">", "gt":
		return value > threshold
	case "<", "lt":
		return value < threshold
	case ">=", "gte":
		return value >= threshold
	case "<=", "lte":
		return value <= threshold
	case "==", "eq":
		return value == threshold
	case "!=", "ne":
		return value != threshold
	default:
		return false
	}
}

// getAlertType converts metric name to alert type
func (nm *NodeMonitor) getAlertType(metric string) AlertType {
	switch metric {
	case "cpu_usage":
		return AlertTypeCPU
	case "memory_usage":
		return AlertTypeMemory
	case "disk_usage":
		return AlertTypeDisk
	case "error_rate":
		return AlertTypePerformance
	case "network_latency":
		return AlertTypeNetwork
	default:
		return AlertTypePerformance
	}
}

// formatAlertMessage formats alert message
func (nm *NodeMonitor) formatAlertMessage(rule *NodeAlertRule, snapshot NodeMetricsSnapshot) string {
	var value float64

	switch rule.Metric {
	case "cpu_usage":
		value = snapshot.Performance.CPUUsage
	case "memory_usage":
		value = snapshot.Performance.MemoryUsage
	case "disk_usage":
		value = snapshot.Performance.DiskUsage
	case "error_rate":
		value = snapshot.Performance.ErrorRate
	case "network_latency":
		value = float64(snapshot.Network.Latency.Nanoseconds()) / 1e6
	default:
		value = 0
	}

	return fmt.Sprintf("%s %.2f exceeds threshold %.2f", rule.Metric, value, rule.Threshold)
}

// runAlertProcessing runs alert processing
func (nm *NodeMonitor) runAlertProcessing(ctx context.Context) {
	for {
		select {
		case alert := <-nm.alertManager.notificationCh:
			nm.processAlert(alert)
		case <-nm.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// processAlert processes an alert
func (nm *NodeMonitor) processAlert(alert NodeAlert) {
	// Store alert
	nm.alertManager.mu.Lock()
	nm.alertManager.activeAlerts[alert.NodeID] = append(nm.alertManager.activeAlerts[alert.NodeID], alert)
	nm.alertManager.mu.Unlock()

	// Add to node
	if node, exists := nm.nodes[alert.NodeID]; exists {
		node.mu.Lock()
		node.Alerts = append(node.Alerts, alert)
		node.mu.Unlock()
	}

	// Send notification (implementation would go here)
}

// runDashboard runs the monitoring dashboard
func (nm *NodeMonitor) runDashboard(ctx context.Context) {
	// Dashboard implementation would go here
	for {
		select {
		case <-nm.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// runAutoScaling runs auto-scaling logic
func (nm *NodeMonitor) runAutoScaling(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			nm.checkAutoScaling()
		case <-nm.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// checkAutoScaling checks if auto-scaling is needed
func (nm *NodeMonitor) checkAutoScaling() {
	for _, node := range nm.nodes {
		if node.Performance.CPUUsage > nm.config.ScalingThreshold {
			nm.triggerScaleUp(node)
		} else if node.Performance.CPUUsage < nm.config.ScalingThreshold/2 {
			nm.triggerScaleDown(node)
		}
	}
}

// triggerScaleUp triggers scale up for a node
func (nm *NodeMonitor) triggerScaleUp(node *MonitoredNode) {
	// Scale up implementation
}

// triggerScaleDown triggers scale down for a node
func (nm *NodeMonitor) triggerScaleDown(node *MonitoredNode) {
	// Scale down implementation
}

// initializeAlertRules initializes default alert rules
func (nm *NodeMonitor) initializeAlertRules() {
	nm.alertManager.rules = []*NodeAlertRule{
		{
			Name:      "High CPU Usage",
			Metric:    "cpu_usage",
			Condition: ">",
			Threshold: 80.0,
			Duration:  5 * time.Minute,
			Severity:  SeverityWarning,
			Enabled:   true,
		},
		{
			Name:      "High Memory Usage",
			Metric:    "memory_usage",
			Condition: ">",
			Threshold: 85.0,
			Duration:  5 * time.Minute,
			Severity:  SeverityWarning,
			Enabled:   true,
		},
		{
			Name:      "High Error Rate",
			Metric:    "error_rate",
			Condition: ">",
			Threshold: 0.1,
			Duration:  2 * time.Minute,
			Severity:  SeverityCritical,
			Enabled:   true,
		},
		{
			Name:      "High Network Latency",
			Metric:    "network_latency",
			Condition: ">",
			Threshold: 100.0, // ms
			Duration:  3 * time.Minute,
			Severity:  SeverityWarning,
			Enabled:   true,
		},
	}
}

// GetClusterMetrics returns cluster-wide metrics
func (nm *NodeMonitor) GetClusterMetrics() *ClusterMetrics {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	metrics := &ClusterMetrics{
		TotalNodes:         len(nm.nodes),
		ActiveNodes:        0,
		FailedNodes:        0,
		AverageCPUUsage:    0.0,
		AverageMemoryUsage: 0.0,
		TotalConnections:   0,
		TotalAlerts:        0,
	}

	for _, node := range nm.nodes {
		if node.Status == NodeStatusActive {
			metrics.ActiveNodes++
		} else if node.Status == NodeStatusFailed {
			metrics.FailedNodes++
		}

		metrics.AverageCPUUsage += node.Performance.CPUUsage
		metrics.AverageMemoryUsage += node.Performance.MemoryUsage
		metrics.TotalConnections += node.Network.ConnectionsActive
		metrics.TotalAlerts += len(node.Alerts)
	}

	if len(nm.nodes) > 0 {
		metrics.AverageCPUUsage /= float64(len(nm.nodes))
		metrics.AverageMemoryUsage /= float64(len(nm.nodes))
	}

	return metrics
}

// ClusterMetrics represents cluster-wide metrics
type ClusterMetrics struct {
	TotalNodes         int     `json:"total_nodes"`
	ActiveNodes        int     `json:"active_nodes"`
	FailedNodes        int     `json:"failed_nodes"`
	AverageCPUUsage    float64 `json:"average_cpu_usage"`
	AverageMemoryUsage float64 `json:"average_memory_usage"`
	TotalConnections   int     `json:"total_connections"`
	TotalAlerts        int     `json:"total_alerts"`
}

// generateAlertID generates unique alert ID
func generateAlertID() string {
	return fmt.Sprintf("alert-%d", time.Now().UnixNano())
}

// Errors
var (
	ErrNodeMonitorAlreadyRunning = &NodeMonitorError{Code: "ALREADY_RUNNING", Message: "node monitor is already running"}
	ErrNodeMonitorNotRunning     = &NodeMonitorError{Code: "NOT_RUNNING", Message: "node monitor is not running"}
	ErrMonitorNodeAlreadyExists  = &NodeMonitorError{Code: "NODE_EXISTS", Message: "node already exists"}
	ErrMonitorNodeNotFound       = &NodeMonitorError{Code: "NODE_NOT_FOUND", Message: "node not found"}
)

// NodeMonitorError represents a node monitor error
type NodeMonitorError struct {
	Code    string
	Message string
}

func (e *NodeMonitorError) Error() string {
	return e.Message + " (" + e.Code + ")"
}

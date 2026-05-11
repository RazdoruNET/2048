package performance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// PerformanceMonitor provides comprehensive performance monitoring
type PerformanceMonitor struct {
	metrics      *MetricsCollector
	alertManager *AlertManager
	dashboard    *Dashboard
	exporter     *MetricsExporter
	config       *MonitoringConfig
	mu           sync.RWMutex
	running      bool
	stopCh       chan struct{}
}

// MetricsCollector collects and aggregates performance metrics
type MetricsCollector struct {
	counters   map[string]*Counter
	gauges     map[string]*Gauge
	histograms map[string]*Histogram
	timers     map[string]*Timer
	mu         sync.RWMutex
}

// AlertManager manages performance alerts
type AlertManager struct {
	rules            []*AlertRule
	alertHistory     []Alert
	notificationChan chan Alert
	mu               sync.RWMutex
}

// Dashboard provides web dashboard for metrics
type Dashboard struct {
	server    *http.Server
	endpoints map[string]http.HandlerFunc
	mu        sync.RWMutex
}

// MetricsExporter exports metrics to external systems
type MetricsExporter struct {
	prometheus      *PrometheusExporter
	influxDB        *InfluxDBExporter
	customExporters map[string]CustomExporter
	mu              sync.RWMutex
}

// MonitoringConfig defines monitoring configuration
type MonitoringConfig struct {
	Enabled           bool          `yaml:"enabled"`
	MetricsInterval   time.Duration `yaml:"metrics_interval"`
	AlertingEnabled   bool          `yaml:"alerting_enabled"`
	DashboardEnabled  bool          `yaml:"dashboard_enabled"`
	DashboardPort     int           `yaml:"dashboard_port"`
	PrometheusEnabled bool          `yaml:"prometheus_enabled"`
	PrometheusPort    int           `yaml:"prometheus_port"`
}

// Counter represents a counter metric
type Counter struct {
	name   string
	value  int64
	labels map[string]string
	mu     sync.RWMutex
}

// Gauge represents a gauge metric
type Gauge struct {
	name   string
	value  float64
	labels map[string]string
	mu     sync.RWMutex
}

// Histogram represents a histogram metric
type Histogram struct {
	name    string
	buckets []float64
	counts  []int64
	sum     float64
	count   int64
	labels  map[string]string
	mu      sync.RWMutex
}

// Timer represents a timer metric
type Timer struct {
	name   string
	values []time.Duration
	labels map[string]string
	mu     sync.RWMutex
}

// AlertRule defines an alert rule
type AlertRule struct {
	Name      string            `yaml:"name"`
	Metric    string            `yaml:"metric"`
	Condition string            `yaml:"condition"`
	Threshold float64           `yaml:"threshold"`
	Duration  time.Duration     `yaml:"duration"`
	Severity  string            `yaml:"severity"`
	Labels    map[string]string `yaml:"labels"`
	Enabled   bool              `yaml:"enabled"`
}

// Alert represents a performance alert
type Alert struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Metric    string            `json:"metric"`
	Value     float64           `json:"value"`
	Threshold float64           `json:"threshold"`
	Severity  string            `json:"severity"`
	Message   string            `json:"message"`
	Timestamp time.Time         `json:"timestamp"`
	Labels    map[string]string `json:"labels"`
	Resolved  bool              `json:"resolved"`
}

// PrometheusExporter exports metrics in Prometheus format
type PrometheusExporter struct {
	metrics map[string]interface{}
	mu      sync.RWMutex
}

// InfluxDBExporter exports metrics to InfluxDB
type InfluxDBExporter struct {
	database string
	server   string
	mu       sync.RWMutex
}

// CustomExporter interface for custom metric exporters
type CustomExporter interface {
	Export(metrics map[string]interface{}) error
	Name() string
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(config *MonitoringConfig) *PerformanceMonitor {
	if config == nil {
		config = &MonitoringConfig{
			Enabled:           true,
			MetricsInterval:   10 * time.Second,
			AlertingEnabled:   true,
			DashboardEnabled:  true,
			DashboardPort:     8080,
			PrometheusEnabled: true,
			PrometheusPort:    9090,
		}
	}

	monitor := &PerformanceMonitor{
		metrics:      NewMetricsCollector(),
		alertManager: NewAlertManager(),
		dashboard:    NewDashboard(config.DashboardPort),
		exporter:     NewMetricsExporter(config),
		config:       config,
		stopCh:       make(chan struct{}),
	}

	return monitor
}

// Start starts the performance monitor
func (pm *PerformanceMonitor) Start() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.running {
		return ErrMonitorAlreadyRunning
	}

	if !pm.config.Enabled {
		return nil
	}

	pm.running = true

	// Start metrics collection
	go pm.collectMetrics()

	// Start alerting
	if pm.config.AlertingEnabled {
		go pm.runAlerting()
	}

	// Start dashboard
	if pm.config.DashboardEnabled {
		go pm.startDashboard()
	}

	// Start exporters
	if pm.config.PrometheusEnabled {
		go pm.startPrometheusExporter()
	}

	return nil
}

// Stop stops the performance monitor
func (pm *PerformanceMonitor) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.running {
		return ErrMonitorNotRunning
	}

	pm.running = false
	close(pm.stopCh)

	// Stop dashboard
	if pm.dashboard != nil {
		pm.dashboard.Stop()
	}

	return nil
}

// collectMetrics collects performance metrics
func (pm *PerformanceMonitor) collectMetrics() {
	ticker := time.NewTicker(pm.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pm.collectSystemMetrics()
			pm.collectApplicationMetrics()
		case <-pm.stopCh:
			return
		}
	}
}

// collectSystemMetrics collects system-level metrics
func (pm *PerformanceMonitor) collectSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Memory metrics
	pm.metrics.SetGauge("memory.alloc_bytes", float64(m.Alloc), nil)
	pm.metrics.SetGauge("memory.total_alloc_bytes", float64(m.TotalAlloc), nil)
	pm.metrics.SetGauge("memory.sys_bytes", float64(m.Sys), nil)
	pm.metrics.SetGauge("memory.num_gc", float64(m.NumGC), nil)
	pm.metrics.SetGauge("memory.gc_cpu_fraction", m.GCCPUFraction, nil)

	// Goroutine metrics
	pm.metrics.SetGauge("goroutines.count", float64(runtime.NumGoroutine()), nil)

	// CPU metrics (simplified)
	pm.metrics.SetGauge("cpu.goroutines", float64(runtime.NumGoroutine()), nil)
}

// collectApplicationMetrics collects application-specific metrics
func (pm *PerformanceMonitor) collectApplicationMetrics() {
	// Connection pool metrics
	pm.metrics.SetGauge("connections.active", float64(pm.getActiveConnections()), nil)
	pm.metrics.SetGauge("connections.pool_size", float64(pm.getPoolSize()), nil)
	pm.metrics.IncrementCounter("connections.created", pm.getConnectionsCreated(), nil)

	// Load balancer metrics
	pm.metrics.SetGauge("load_balancer.endpoints", float64(pm.getEndpointCount()), nil)
	pm.metrics.SetGauge("load_balancer.healthy_endpoints", float64(pm.getHealthyEndpointCount()), nil)
	pm.metrics.IncrementCounter("load_balancer.requests", pm.getTotalRequests(), nil)

	// Performance metrics
	for _, duration := range pm.getRequestDurations() {
		pm.metrics.RecordHistogram("request.duration", duration, nil)
	}
	for _, responseTime := range pm.getResponseTimes() {
		pm.metrics.RecordTimer("response.time", responseTime, nil)
	}
}

// runAlerting runs the alerting system
func (pm *PerformanceMonitor) runAlerting() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pm.checkAlerts()
		case <-pm.stopCh:
			return
		}
	}
}

// checkAlerts checks all alert rules
func (pm *PerformanceMonitor) checkAlerts() {
	for _, rule := range pm.alertManager.GetRules() {
		if !rule.Enabled {
			continue
		}

		value := pm.metrics.GetMetric(rule.Metric)
		if value == nil {
			continue
		}

		if pm.evaluateCondition(value, rule.Condition, rule.Threshold) {
			alert := Alert{
				ID:        fmt.Sprintf("%s-%d", rule.Name, time.Now().Unix()),
				Name:      rule.Name,
				Metric:    rule.Metric,
				Value:     value.(float64),
				Threshold: rule.Threshold,
				Severity:  rule.Severity,
				Message:   fmt.Sprintf("Metric %s %.2f exceeds threshold %.2f", rule.Metric, value.(float64), rule.Threshold),
				Timestamp: time.Now(),
				Labels:    rule.Labels,
				Resolved:  false,
			}

			pm.alertManager.TriggerAlert(alert)
		}
	}
}

// evaluateCondition evaluates alert condition
func (pm *PerformanceMonitor) evaluateCondition(value interface{}, condition string, threshold float64) bool {
	val, ok := value.(float64)
	if !ok {
		return false
	}

	switch condition {
	case ">", "gt":
		return val > threshold
	case "<", "lt":
		return val < threshold
	case ">=", "gte":
		return val >= threshold
	case "<=", "lte":
		return val <= threshold
	case "==", "eq":
		return val == threshold
	case "!=", "ne":
		return val != threshold
	default:
		return false
	}
}

// startDashboard starts the web dashboard
func (pm *PerformanceMonitor) startDashboard() {
	pm.dashboard.Start()
}

// startPrometheusExporter starts the Prometheus exporter
func (pm *PerformanceMonitor) startPrometheusExporter() {
	pm.exporter.StartPrometheus()
}

// Helper methods for getting metrics (placeholders)
func (pm *PerformanceMonitor) getActiveConnections() int64       { return 0 }
func (pm *PerformanceMonitor) getPoolSize() int64                { return 0 }
func (pm *PerformanceMonitor) getConnectionsCreated() int64      { return 0 }
func (pm *PerformanceMonitor) getEndpointCount() int64           { return 0 }
func (pm *PerformanceMonitor) getHealthyEndpointCount() int64    { return 0 }
func (pm *PerformanceMonitor) getTotalRequests() int64           { return 0 }
func (pm *PerformanceMonitor) getRequestDurations() []float64    { return []float64{} }
func (pm *PerformanceMonitor) getResponseTimes() []time.Duration { return []time.Duration{} }

// GetMetrics returns current performance metrics
func (pm *PerformanceMonitor) GetMetrics() *PerformanceMetrics {
	pm.metrics.mu.RLock()
	defer pm.metrics.mu.RUnlock()

	// Get metrics from the collector
	memoryUsage := pm.metrics.GetMetric("memory.alloc_bytes")
	cpuUsage := pm.metrics.GetMetric("cpu.goroutines")
	goroutines := pm.metrics.GetMetric("goroutines.count")
	gcFrequency := pm.metrics.GetMetric("memory.num_gc")

	return &PerformanceMetrics{
		MemoryUsage:  getInt64Value(memoryUsage),
		CPUUsage:     getFloat64Value(cpuUsage),
		Goroutines:   getInt(goroutines),
		GCFrequency:  getInt64Value(gcFrequency),
		IOOperations: 0,
		Latency:      0,
		Throughput:   0,
		ErrorRate:    0,
	}
}

// Helper functions for type conversion
func getInt64Value(value interface{}) int64 {
	if v, ok := value.(int64); ok {
		return v
	}
	if v, ok := value.(float64); ok {
		return int64(v)
	}
	return 0
}

func getFloat64Value(value interface{}) float64 {
	if v, ok := value.(float64); ok {
		return v
	}
	if v, ok := value.(int64); ok {
		return float64(v)
	}
	return 0
}

func getInt(value interface{}) int {
	if v, ok := value.(int); ok {
		return v
	}
	if v, ok := value.(int64); ok {
		return int(v)
	}
	if v, ok := value.(float64); ok {
		return int(v)
	}
	return 0
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		counters:   make(map[string]*Counter),
		gauges:     make(map[string]*Gauge),
		histograms: make(map[string]*Histogram),
		timers:     make(map[string]*Timer),
	}
}

// NewAlertManager creates a new alert manager
func NewAlertManager() *AlertManager {
	return &AlertManager{
		rules:            []*AlertRule{},
		alertHistory:     []Alert{},
		notificationChan: make(chan Alert, 100),
	}
}

// NewDashboard creates a new dashboard
func NewDashboard(port int) *Dashboard {
	return &Dashboard{
		server: &http.Server{
			Addr: fmt.Sprintf(":%d", port),
		},
		endpoints: make(map[string]http.HandlerFunc),
	}
}

// NewMetricsExporter creates a new metrics exporter
func NewMetricsExporter(config *MonitoringConfig) *MetricsExporter {
	return &MetricsExporter{
		prometheus:      NewPrometheusExporter(),
		influxDB:        NewInfluxDBExporter(),
		customExporters: make(map[string]CustomExporter),
	}
}

// MetricsCollector methods
func (mc *MetricsCollector) IncrementCounter(name string, value int64, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if counter, exists := mc.counters[name]; exists {
		counter.Increment(value)
	} else {
		mc.counters[name] = NewCounter(name, labels)
		mc.counters[name].Increment(value)
	}
}

func (mc *MetricsCollector) SetGauge(name string, value float64, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if gauge, exists := mc.gauges[name]; exists {
		gauge.Set(value)
	} else {
		mc.gauges[name] = NewGauge(name, labels)
		mc.gauges[name].Set(value)
	}
}

func (mc *MetricsCollector) RecordHistogram(name string, value float64, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if histogram, exists := mc.histograms[name]; exists {
		histogram.Observe(value)
	} else {
		mc.histograms[name] = NewHistogram(name, []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0}, labels)
		mc.histograms[name].Observe(value)
	}
}

func (mc *MetricsCollector) RecordTimer(name string, value time.Duration, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if timer, exists := mc.timers[name]; exists {
		timer.Record(value)
	} else {
		mc.timers[name] = NewTimer(name, labels)
		mc.timers[name].Record(value)
	}
}

func (mc *MetricsCollector) GetMetric(name string) interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if counter, exists := mc.counters[name]; exists {
		return counter.Value()
	}

	if gauge, exists := mc.gauges[name]; exists {
		return gauge.Value()
	}

	if histogram, exists := mc.histograms[name]; exists {
		return histogram.Sum()
	}

	if timer, exists := mc.timers[name]; exists {
		return timer.Average()
	}

	return nil
}

// AlertManager methods
func (am *AlertManager) AddRule(rule *AlertRule) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.rules = append(am.rules, rule)
}

func (am *AlertManager) GetRules() []*AlertRule {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.rules
}

func (am *AlertManager) TriggerAlert(alert Alert) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.alertHistory = append(am.alertHistory, alert)
	select {
	case am.notificationChan <- alert:
	default:
		// Channel full, drop alert
	}
}

// Dashboard methods
func (d *Dashboard) Start() error {
	d.setupRoutes()
	return d.server.ListenAndServe()
}

func (d *Dashboard) Stop() error {
	return d.server.Close()
}

func (d *Dashboard) setupRoutes() {
	d.endpoints["/metrics"] = d.metricsHandler
	d.endpoints["/health"] = d.healthHandler
	d.endpoints["/alerts"] = d.alertsHandler

	for path, handler := range d.endpoints {
		http.HandleFunc(path, handler)
	}
}

func (d *Dashboard) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"time":   time.Now(),
	})
}

func (d *Dashboard) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
	})
}

func (d *Dashboard) alertsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"alerts": []Alert{},
	})
}

// MetricsExporter methods
func (me *MetricsExporter) StartPrometheus() {
	// Prometheus exporter implementation
}

// Constructor functions for metric types
func NewCounter(name string, labels map[string]string) *Counter {
	return &Counter{
		name:   name,
		labels: labels,
	}
}

func NewGauge(name string, labels map[string]string) *Gauge {
	return &Gauge{
		name:   name,
		labels: labels,
	}
}

func NewHistogram(name string, buckets []float64, labels map[string]string) *Histogram {
	return &Histogram{
		name:    name,
		buckets: buckets,
		counts:  make([]int64, len(buckets)+1),
		labels:  labels,
	}
}

func NewTimer(name string, labels map[string]string) *Timer {
	return &Timer{
		name:   name,
		labels: labels,
	}
}

// Counter methods
func (c *Counter) Increment(value int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value += value
}

func (c *Counter) Value() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.value
}

// Gauge methods
func (g *Gauge) Set(value float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value = value
}

func (g *Gauge) Value() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.value
}

// Histogram methods
func (h *Histogram) Observe(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.sum += value
	h.count++

	for i, bucket := range h.buckets {
		if value <= bucket {
			h.counts[i]++
			return
		}
	}
	h.counts[len(h.buckets)]++
}

func (h *Histogram) Sum() float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.sum
}

// Timer methods
func (t *Timer) Record(value time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.values = append(t.values, value)
}

func (t *Timer) Average() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.values) == 0 {
		return 0
	}

	var sum time.Duration
	for _, v := range t.values {
		sum += v
	}
	return sum / time.Duration(len(t.values))
}

// Helper constructors
func NewPrometheusExporter() *PrometheusExporter {
	return &PrometheusExporter{
		metrics: make(map[string]interface{}),
	}
}

func NewInfluxDBExporter() *InfluxDBExporter {
	return &InfluxDBExporter{
		database: "performance",
		server:   "http://localhost:8086",
	}
}

// Errors
var (
	ErrMonitorAlreadyRunning = &MonitorError{Code: "ALREADY_RUNNING", Message: "monitor is already running"}
	ErrMonitorNotRunning     = &MonitorError{Code: "NOT_RUNNING", Message: "monitor is not running"}
)

// MonitorError represents a monitor error
type MonitorError struct {
	Code    string
	Message string
}

func (e *MonitorError) Error() string {
	return e.Message + " (" + e.Code + ")"
}

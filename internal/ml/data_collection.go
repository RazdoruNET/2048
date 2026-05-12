package ml

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// DataCollector manages collection of ML training data
type DataCollector struct {
	mu              sync.RWMutex
	feedbackBuffer  []FeedbackEntry
	requestBuffer   []RequestEntry
	performanceData []PerformanceEntry
	maxBufferSize   int
	flushInterval   time.Duration
	ctx             context.Context
	cancel          context.CancelFunc
	processors      []DataProcessor
	exporters       []DataExporter
}

// RequestEntry represents a request for data collection
type RequestEntry struct {
	ID           string            `json:"id"`
	Domain       string            `json:"domain"`
	Technique    string            `json:"technique"`
	Success      bool              `json:"success"`
	Latency      time.Duration     `json:"latency"`
	BytesIn      int64             `json:"bytes_in"`
	BytesOut     int64             `json:"bytes_out"`
	Timestamp    time.Time         `json:"timestamp"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// PerformanceEntry represents performance metrics
type PerformanceEntry struct {
	Timestamp    time.Time         `json:"timestamp"`
	CPUUsage     float64           `json:"cpu_usage"`
	MemoryUsage  int64             `json:"memory_usage"`
	Connections  int               `json:"connections"`
	Throughput   float64           `json:"throughput"`
	ErrorRate    float64           `json:"error_rate"`
	ResponseTime time.Duration     `json:"response_time"`
}

// DataProcessor processes collected data
type DataProcessor interface {
	ProcessFeedback([]FeedbackEntry) ([]FeedbackEntry, error)
	ProcessRequests([]RequestEntry) ([]RequestEntry, error)
	ProcessPerformance([]PerformanceEntry) ([]PerformanceEntry, error)
}

// DataExporter exports processed data
type DataExporter interface {
	ExportFeedback([]FeedbackEntry) error
	ExportRequests([]RequestEntry) error
	ExportPerformance([]PerformanceEntry) error
}

// NewDataCollector creates a new data collector
func NewDataCollector(maxBufferSize int, flushInterval time.Duration) *DataCollector {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &DataCollector{
		feedbackBuffer:  make([]FeedbackEntry, 0),
		requestBuffer:   make([]RequestEntry, 0),
		performanceData: make([]PerformanceEntry, 0),
		maxBufferSize:   maxBufferSize,
		flushInterval:   flushInterval,
		ctx:             ctx,
		cancel:          cancel,
		processors:      make([]DataProcessor, 0),
		exporters:       make([]DataExporter, 0),
	}
}

// Start starts the data collector
func (dc *DataCollector) Start() {
	go dc.collectionLoop()
	log.Printf("Data collector started with buffer size %d and flush interval %v", dc.maxBufferSize, dc.flushInterval)
}

// Stop stops the data collector
func (dc *DataCollector) Stop() {
	dc.cancel()
	
	// Flush remaining data
	dc.flushAllBuffers()
	log.Println("Data collector stopped")
}

// collectionLoop runs the periodic data collection and flushing
func (dc *DataCollector) collectionLoop() {
	ticker := time.NewTicker(dc.flushInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-dc.ctx.Done():
			return
		case <-ticker.C:
			dc.flushAllBuffers()
		}
	}
}

// CollectFeedback collects feedback data
func (dc *DataCollector) CollectFeedback(domain, technique string, success bool) error {
	entry := FeedbackEntry{
		Domain:       domain,
		Technique:    technique,
		Success:      success,
		Timestamp:    time.Now(),
		Effectiveness: dc.calculateEffectiveness(success),
	}
	
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	dc.feedbackBuffer = append(dc.feedbackBuffer, entry)
	
	// Check if buffer needs flushing
	if len(dc.feedbackBuffer) >= dc.maxBufferSize {
		go dc.flushFeedbackBuffer()
	}
	
	return nil
}

// CollectRequest collects request data
func (dc *DataCollector) CollectRequest(id, domain, technique string, success bool, latency time.Duration, bytesIn, bytesOut int64, metadata map[string]interface{}) error {
	entry := RequestEntry{
		ID:        id,
		Domain:    domain,
		Technique: technique,
		Success:   success,
		Latency:   latency,
		BytesIn:   bytesIn,
		BytesOut:  bytesOut,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}
	
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	dc.requestBuffer = append(dc.requestBuffer, entry)
	
	// Check if buffer needs flushing
	if len(dc.requestBuffer) >= dc.maxBufferSize {
		go dc.flushRequestBuffer()
	}
	
	return nil
}

// CollectPerformance collects performance metrics
func (dc *DataCollector) CollectPerformance(cpuUsage float64, memoryUsage int64, connections int, throughput, errorRate float64, responseTime time.Duration) error {
	entry := PerformanceEntry{
		Timestamp:    time.Now(),
		CPUUsage:     cpuUsage,
		MemoryUsage:  memoryUsage,
		Connections:  connections,
		Throughput:   throughput,
		ErrorRate:    errorRate,
		ResponseTime: responseTime,
	}
	
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	dc.performanceData = append(dc.performanceData, entry)
	
	// Check if buffer needs flushing
	if len(dc.performanceData) >= dc.maxBufferSize {
		go dc.flushPerformanceBuffer()
	}
	
	return nil
}

// calculateEffectiveness calculates effectiveness score
func (dc *DataCollector) calculateEffectiveness(success bool) float64 {
	if success {
		return 1.0
	}
	return 0.0
}

// flushAllBuffers flushes all data buffers
func (dc *DataCollector) flushAllBuffers() {
	dc.mu.Lock()
	
	feedbackData := make([]FeedbackEntry, len(dc.feedbackBuffer))
	copy(feedbackData, dc.feedbackBuffer)
	dc.feedbackBuffer = dc.feedbackBuffer[:0]
	
	requestData := make([]RequestEntry, len(dc.requestBuffer))
	copy(requestData, dc.requestBuffer)
	dc.requestBuffer = dc.requestBuffer[:0]
	
	performanceData := make([]PerformanceEntry, len(dc.performanceData))
	copy(performanceData, dc.performanceData)
	dc.performanceData = dc.performanceData[:0]
	
	dc.mu.Unlock()
	
	// Process and export data in parallel
	if len(feedbackData) > 0 {
		go dc.processAndExportFeedback(feedbackData)
	}
	
	if len(requestData) > 0 {
		go dc.processAndExportRequests(requestData)
	}
	
	if len(performanceData) > 0 {
		go dc.processAndExportPerformance(performanceData)
	}
}

// flushFeedbackBuffer flushes the feedback buffer
func (dc *DataCollector) flushFeedbackBuffer() {
	dc.mu.Lock()
	data := make([]FeedbackEntry, len(dc.feedbackBuffer))
	copy(data, dc.feedbackBuffer)
	dc.feedbackBuffer = dc.feedbackBuffer[:0]
	dc.mu.Unlock()
	
	dc.processAndExportFeedback(data)
}

// flushRequestBuffer flushes the request buffer
func (dc *DataCollector) flushRequestBuffer() {
	dc.mu.Lock()
	data := make([]RequestEntry, len(dc.requestBuffer))
	copy(data, dc.requestBuffer)
	dc.requestBuffer = dc.requestBuffer[:0]
	dc.mu.Unlock()
	
	dc.processAndExportRequests(data)
}

// flushPerformanceBuffer flushes the performance buffer
func (dc *DataCollector) flushPerformanceBuffer() {
	dc.mu.Lock()
	data := make([]PerformanceEntry, len(dc.performanceData))
	copy(data, dc.performanceData)
	dc.performanceData = dc.performanceData[:0]
	dc.mu.Unlock()
	
	dc.processAndExportPerformance(data)
}

// processAndExportFeedback processes and exports feedback data
func (dc *DataCollector) processAndExportFeedback(data []FeedbackEntry) {
	processedData := data
	
	// Apply processors
	for _, processor := range dc.processors {
		var err error
		processedData, err = processor.ProcessFeedback(processedData)
		if err != nil {
			log.Printf("Feedback processor error: %v", err)
			continue
		}
	}
	
	// Export data
	for _, exporter := range dc.exporters {
		if err := exporter.ExportFeedback(processedData); err != nil {
			log.Printf("Feedback export error: %v", err)
		}
	}
}

// processAndExportRequests processes and exports request data
func (dc *DataCollector) processAndExportRequests(data []RequestEntry) {
	processedData := data
	
	// Apply processors
	for _, processor := range dc.processors {
		var err error
		processedData, err = processor.ProcessRequests(processedData)
		if err != nil {
			log.Printf("Request processor error: %v", err)
			continue
		}
	}
	
	// Export data
	for _, exporter := range dc.exporters {
		if err := exporter.ExportRequests(processedData); err != nil {
			log.Printf("Request export error: %v", err)
		}
	}
}

// processAndExportPerformance processes and exports performance data
func (dc *DataCollector) processAndExportPerformance(data []PerformanceEntry) {
	processedData := data
	
	// Apply processors
	for _, processor := range dc.processors {
		var err error
		processedData, err = processor.ProcessPerformance(processedData)
		if err != nil {
			log.Printf("Performance processor error: %v", err)
			continue
		}
	}
	
	// Export data
	for _, exporter := range dc.exporters {
		if err := exporter.ExportPerformance(processedData); err != nil {
			log.Printf("Performance export error: %v", err)
		}
	}
}

// AddProcessor adds a data processor
func (dc *DataCollector) AddProcessor(processor DataProcessor) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	dc.processors = append(dc.processors, processor)
}

// RemoveProcessor removes a data processor
func (dc *DataCollector) RemoveProcessor(processorToRemove DataProcessor) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	processors := make([]DataProcessor, 0, len(dc.processors))
	for _, processor := range dc.processors {
		if fmt.Sprintf("%p", processor) != fmt.Sprintf("%p", processorToRemove) {
			processors = append(processors, processor)
		}
	}
	dc.processors = processors
}

// AddExporter adds a data exporter
func (dc *DataCollector) AddExporter(exporter DataExporter) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	dc.exporters = append(dc.exporters, exporter)
}

// RemoveExporter removes a data exporter
func (dc *DataCollector) RemoveExporter(exporterToRemove DataExporter) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	exporters := make([]DataExporter, 0, len(dc.exporters))
	for _, exporter := range dc.exporters {
		if fmt.Sprintf("%p", exporter) != fmt.Sprintf("%p", exporterToRemove) {
			exporters = append(exporters, exporter)
		}
	}
	dc.exporters = exporters
}

// GetStatistics returns data collection statistics
func (dc *DataCollector) GetStatistics() map[string]interface{} {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	
	stats := make(map[string]interface{})
	
	stats["feedback_buffer_size"] = len(dc.feedbackBuffer)
	stats["request_buffer_size"] = len(dc.requestBuffer)
	stats["performance_buffer_size"] = len(dc.performanceData)
	stats["max_buffer_size"] = dc.maxBufferSize
	stats["flush_interval"] = dc.flushInterval.String()
	stats["processors_count"] = len(dc.processors)
	stats["exporters_count"] = len(dc.exporters)
	
	return stats
}

// GetBufferData returns current buffer data
func (dc *DataCollector) GetBufferData() map[string]interface{} {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	
	data := make(map[string]interface{})
	
	// Return copies to prevent external modification
	feedbackCopy := make([]FeedbackEntry, len(dc.feedbackBuffer))
	copy(feedbackCopy, dc.feedbackBuffer)
	data["feedback"] = feedbackCopy
	
	requestCopy := make([]RequestEntry, len(dc.requestBuffer))
	copy(requestCopy, dc.requestBuffer)
	data["requests"] = requestCopy
	
	performanceCopy := make([]PerformanceEntry, len(dc.performanceData))
	copy(performanceCopy, dc.performanceData)
	data["performance"] = performanceCopy
	
	return data
}

// ClearBuffers clears all data buffers
func (dc *DataCollector) ClearBuffers() {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	dc.feedbackBuffer = dc.feedbackBuffer[:0]
	dc.requestBuffer = dc.requestBuffer[:0]
	dc.performanceData = dc.performanceData[:0]
	
	log.Println("Data buffers cleared")
}

// DataCleaner implements DataProcessor for data cleaning
type DataCleaner struct {
	maxAge time.Duration
}

// NewDataCleaner creates a new data cleaner
func NewDataCleaner(maxAge time.Duration) *DataCleaner {
	return &DataCleaner{
		maxAge: maxAge,
	}
}

// ProcessFeedback processes feedback data
func (dc *DataCleaner) ProcessFeedback(data []FeedbackEntry) ([]FeedbackEntry, error) {
	cutoff := time.Now().Add(-dc.maxAge)
	
	var cleaned []FeedbackEntry
	for _, entry := range data {
		if entry.Timestamp.After(cutoff) {
			cleaned = append(cleaned, entry)
		}
	}
	
	return cleaned, nil
}

// ProcessRequests processes request data
func (dc *DataCleaner) ProcessRequests(data []RequestEntry) ([]RequestEntry, error) {
	cutoff := time.Now().Add(-dc.maxAge)
	
	var cleaned []RequestEntry
	for _, entry := range data {
		if entry.Timestamp.After(cutoff) {
			cleaned = append(cleaned, entry)
		}
	}
	
	return cleaned, nil
}

// ProcessPerformance processes performance data
func (dc *DataCleaner) ProcessPerformance(data []PerformanceEntry) ([]PerformanceEntry, error) {
	cutoff := time.Now().Add(-dc.maxAge)
	
	var cleaned []PerformanceEntry
	for _, entry := range data {
		if entry.Timestamp.After(cutoff) {
			cleaned = append(cleaned, entry)
		}
	}
	
	return cleaned, nil
}

// FileExporter implements DataExporter for file-based export
type FileExporter struct {
	feedbackPath  string
	requestPath   string
	performancePath string
}

// NewFileExporter creates a new file exporter
func NewFileExporter(feedbackPath, requestPath, performancePath string) *FileExporter {
	return &FileExporter{
		feedbackPath:     feedbackPath,
		requestPath:      requestPath,
		performancePath: performancePath,
	}
}

// ExportFeedback exports feedback data to file
func (fe *FileExporter) ExportFeedback(data []FeedbackEntry) error {
	// Implementation would write data to file
	// For now, just log the export
	log.Printf("Exported %d feedback entries to %s", len(data), fe.feedbackPath)
	return nil
}

// ExportRequests exports request data to file
func (fe *FileExporter) ExportRequests(data []RequestEntry) error {
	// Implementation would write data to file
	log.Printf("Exported %d request entries to %s", len(data), fe.requestPath)
	return nil
}

// ExportPerformance exports performance data to file
func (fe *FileExporter) ExportPerformance(data []PerformanceEntry) error {
	// Implementation would write data to file
	log.Printf("Exported %d performance entries to %s", len(data), fe.performancePath)
	return nil
}

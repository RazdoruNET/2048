package ml

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// HotReloadManager manages hot reloading of ML configurations
type HotReloadManager struct {
	mu              sync.RWMutex
	engine          MLEngine
	currentConfig   *MLConfig
	configHistory   []ConfigHistoryEntry
	reloadCallbacks []ReloadCallback
	watchers        []ConfigWatcher
	ctx             context.Context
	cancel          context.CancelFunc
	enabled         bool
}

// ConfigHistoryEntry represents a configuration change in history
type ConfigHistoryEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Config      *MLConfig `json:"config"`
	Reason      string    `json:"reason"`
	Success     bool      `json:"success"`
	Error       string    `json:"error,omitempty"`
	AppliedBy   string    `json:"applied_by"`
}

// ReloadCallback is called when configuration is reloaded
type ReloadCallback func(oldConfig, newConfig *MLConfig) error

// ConfigWatcher watches for configuration changes
type ConfigWatcher interface {
	Watch(ctx context.Context, callback func(*MLConfig) error) error
	Stop() error
}

// NewHotReloadManager creates a new hot reload manager
func NewHotReloadManager(engine MLEngine) *HotReloadManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &HotReloadManager{
		engine:          engine,
		configHistory:   make([]ConfigHistoryEntry, 0),
		reloadCallbacks: make([]ReloadCallback, 0),
		watchers:        make([]ConfigWatcher, 0),
		ctx:             ctx,
		cancel:          cancel,
		enabled:         true,
	}
}

// Enable enables hot reloading
func (hrm *HotReloadManager) Enable() {
	hrm.mu.Lock()
	defer hrm.mu.Unlock()
	hrm.enabled = true
	log.Println("Hot reload manager enabled")
}

// Disable disables hot reloading
func (hrm *HotReloadManager) Disable() {
	hrm.mu.Lock()
	defer hrm.mu.Unlock()
	hrm.enabled = false
	log.Println("Hot reload manager disabled")
}

// IsEnabled returns whether hot reloading is enabled
func (hrm *HotReloadManager) IsEnabled() bool {
	hrm.mu.RLock()
	defer hrm.mu.RUnlock()
	return hrm.enabled
}

// AddReloadCallback adds a callback to be called on configuration reload
func (hrm *HotReloadManager) AddReloadCallback(callback ReloadCallback) {
	hrm.mu.Lock()
	defer hrm.mu.Unlock()
	hrm.reloadCallbacks = append(hrm.reloadCallbacks, callback)
}

// RemoveReloadCallback removes a reload callback
func (hrm *HotReloadManager) RemoveReloadCallback(callbackToRemove ReloadCallback) {
	hrm.mu.Lock()
	defer hrm.mu.Unlock()
	
	callbacks := make([]ReloadCallback, 0, len(hrm.reloadCallbacks))
	for _, callback := range hrm.reloadCallbacks {
		// Compare function pointers (this is a simple approach)
		if fmt.Sprintf("%p", callback) != fmt.Sprintf("%p", callbackToRemove) {
			callbacks = append(callbacks, callback)
		}
	}
	hrm.reloadCallbacks = callbacks
}

// AddWatcher adds a configuration watcher
func (hrm *HotReloadManager) AddWatcher(watcher ConfigWatcher) error {
	hrm.mu.Lock()
	defer hrm.mu.Unlock()
	
	if !hrm.enabled {
		return fmt.Errorf("hot reload is disabled")
	}
	
	// Start the watcher
	go func() {
		err := watcher.Watch(hrm.ctx, func(config *MLConfig) error {
			return hrm.ReloadConfig(config, "watcher")
		})
		if err != nil {
			log.Printf("Configuration watcher error: %v", err)
		}
	}()
	
	hrm.watchers = append(hrm.watchers, watcher)
	log.Printf("Added configuration watcher: %T", watcher)
	
	return nil
}

// RemoveWatcher removes a configuration watcher
func (hrm *HotReloadManager) RemoveWatcher(watcher ConfigWatcher) error {
	hrm.mu.Lock()
	defer hrm.mu.Unlock()
	
	// Stop the watcher
	if err := watcher.Stop(); err != nil {
		return fmt.Errorf("failed to stop watcher: %v", err)
	}
	
	// Remove from list
	watchers := make([]ConfigWatcher, 0, len(hrm.watchers))
	for _, w := range hrm.watchers {
		if w != watcher {
			watchers = append(watchers, w)
		}
	}
	hrm.watchers = watchers
	
	log.Printf("Removed configuration watcher: %T", watcher)
	return nil
}

// ReloadConfig reloads the configuration with validation and callbacks
func (hrm *HotReloadManager) ReloadConfig(newConfig *MLConfig, reason string) error {
	hrm.mu.Lock()
	defer hrm.mu.Unlock()
	
	if !hrm.enabled {
		return fmt.Errorf("hot reload is disabled")
	}
	
	// Validate new configuration
	validator := NewConfigValidator()
	validationResult := validator.ValidateConfig(newConfig)
	
	if !validationResult.Valid {
		return fmt.Errorf("configuration validation failed: %v", validationResult.Errors)
	}
	
	// Get current config
	oldConfig := hrm.currentConfig
	if oldConfig == nil {
		// Load current config from engine
		oldConfig = hrm.engine.GetConfig()
	}
	
	// Check if configuration actually changed
	if hrm.configsEqual(oldConfig, newConfig) {
		log.Println("Configuration unchanged, skipping reload")
		return nil
	}
	
	// Execute reload callbacks
	for i, callback := range hrm.reloadCallbacks {
		if err := callback(oldConfig, newConfig); err != nil {
			log.Printf("Reload callback %d failed: %v", i, err)
			// Continue with other callbacks
		}
	}
	
	// Apply new configuration
	if err := hrm.engine.UpdateConfig(newConfig); err != nil {
		// Add to history as failed
		hrm.addToHistory(oldConfig, newConfig, reason, false, err.Error(), "system")
		return fmt.Errorf("failed to apply configuration: %v", err)
	}
	
	// Update current config
	hrm.currentConfig = newConfig
	
	// Add to history as successful
	hrm.addToHistory(oldConfig, newConfig, reason, true, "", "system")
	
	log.Printf("Configuration reloaded successfully (reason: %s)", reason)
	return nil
}

// configsEqual checks if two configurations are equal
func (hrm *HotReloadManager) configsEqual(config1, config2 *MLConfig) bool {
	if config1 == nil && config2 == nil {
		return true
	}
	if config1 == nil || config2 == nil {
		return false
	}
	
	// Compare basic fields
	if config1.Enabled != config2.Enabled ||
		config1.ModelPath != config2.ModelPath ||
		config1.UpdateInterval != config2.UpdateInterval ||
		config1.MaxRetries != config2.MaxRetries ||
		config1.Timeout != config2.Timeout {
		return false
	}
	
	// Compare custom parameters
	return hrm.mapsEqual(config1.CustomParameters, config2.CustomParameters)
}

// mapsEqual checks if two maps are equal
func (hrm *HotReloadManager) mapsEqual(map1, map2 map[string]interface{}) bool {
	if len(map1) != len(map2) {
		return false
	}
	
	for key, value1 := range map1 {
		value2, exists := map2[key]
		if !exists || !hrm.valuesEqual(value1, value2) {
			return false
		}
	}
	
	return true
}

// valuesEqual checks if two values are equal
func (hrm *HotReloadManager) valuesEqual(value1, value2 interface{}) bool {
	// Simple equality check for most types
	return fmt.Sprintf("%v", value1) == fmt.Sprintf("%v", value2)
}

// addToHistory adds a configuration change to history
func (hrm *HotReloadManager) addToHistory(oldConfig, newConfig *MLConfig, reason string, success bool, errorMsg, appliedBy string) {
	entry := ConfigHistoryEntry{
		Timestamp: time.Now(),
		Config:    newConfig,
		Reason:    reason,
		Success:   success,
		Error:     errorMsg,
		AppliedBy: appliedBy,
	}
	
	hrm.configHistory = append(hrm.configHistory, entry)
	
	// Keep only last 100 entries
	if len(hrm.configHistory) > 100 {
		hrm.configHistory = hrm.configHistory[1:]
	}
}

// GetConfigHistory returns the configuration change history
func (hrm *HotReloadManager) GetConfigHistory() []ConfigHistoryEntry {
	hrm.mu.RLock()
	defer hrm.mu.RUnlock()
	
	// Return a copy to prevent external modification
	history := make([]ConfigHistoryEntry, len(hrm.configHistory))
	copy(history, hrm.configHistory)
	
	return history
}

// GetCurrentConfig returns the current configuration
func (hrm *HotReloadManager) GetCurrentConfig() *MLConfig {
	hrm.mu.RLock()
	defer hrm.mu.RUnlock()
	
	if hrm.currentConfig == nil {
		return hrm.engine.GetConfig()
	}
	
	// Return a copy to prevent external modification
	configCopy := *hrm.currentConfig
	return &configCopy
}

// GetStatistics returns hot reload statistics
func (hrm *HotReloadManager) GetStatistics() map[string]interface{} {
	hrm.mu.RLock()
	defer hrm.mu.RUnlock()
	
	stats := make(map[string]interface{})
	
	// Basic stats
	stats["enabled"] = hrm.enabled
	stats["watchers_count"] = len(hrm.watchers)
	stats["callbacks_count"] = len(hrm.reloadCallbacks)
	stats["history_size"] = len(hrm.configHistory)
	
	// History stats
	successCount := 0
	failureCount := 0
	
	for _, entry := range hrm.configHistory {
		if entry.Success {
			successCount++
		} else {
			failureCount++
		}
	}
	
	stats["successful_reloads"] = successCount
	stats["failed_reloads"] = failureCount
	
	if len(hrm.configHistory) > 0 {
		stats["success_rate"] = float64(successCount) / float64(len(hrm.configHistory)) * 100
		stats["last_reload"] = hrm.configHistory[len(hrm.configHistory)-1].Timestamp
	}
	
	return stats
}

// RollbackToConfig rolls back to a previous configuration
func (hrm *HotReloadManager) RollbackToConfig(timestamp time.Time) error {
	hrm.mu.Lock()
	defer hrm.mu.Unlock()
	
	// Find the configuration in history
	for _, entry := range hrm.configHistory {
		if entry.Timestamp.Equal(timestamp) && entry.Success {
			return hrm.ReloadConfig(entry.Config, fmt.Sprintf("rollback to %s", timestamp))
		}
	}
	
	return fmt.Errorf("configuration not found in history for timestamp: %s", timestamp)
}

// Stop stops the hot reload manager
func (hrm *HotReloadManager) Stop() {
	hrm.cancel()
	
	hrm.mu.Lock()
	defer hrm.mu.Unlock()
	
	// Stop all watchers
	for _, watcher := range hrm.watchers {
		if err := watcher.Stop(); err != nil {
			log.Printf("Failed to stop watcher: %v", err)
		}
	}
	
	hrm.watchers = make([]ConfigWatcher, 0)
	hrm.enabled = false
	
	log.Println("Hot reload manager stopped")
}

// FileConfigWatcher implements ConfigWatcher for file-based configuration
type FileConfigWatcher struct {
	filePath string
	lastMod  time.Time
	stopCh   chan struct{}
}

// NewFileConfigWatcher creates a new file configuration watcher
func NewFileConfigWatcher(filePath string) *FileConfigWatcher {
	return &FileConfigWatcher{
		filePath: filePath,
		stopCh:   make(chan struct{}),
	}
}

// Watch watches for file changes
func (fcw *FileConfigWatcher) Watch(ctx context.Context, callback func(*MLConfig) error) error {
	ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-fcw.stopCh:
			return fmt.Errorf("watcher stopped")
		case <-ticker.C:
			if err := fcw.checkFileChange(callback); err != nil {
				log.Printf("File change check error: %v", err)
			}
		}
	}
}

// checkFileChange checks if the file has changed
func (fcw *FileConfigWatcher) checkFileChange(callback func(*MLConfig) error) error {
	// This is a placeholder implementation
	// In a real implementation, you would:
	// 1. Check file modification time
	// 2. Read and parse the file
	// 3. Call the callback with the new config
	
	// For now, we'll just return nil
	return nil
}

// Stop stops the file watcher
func (fcw *FileConfigWatcher) Stop() error {
	close(fcw.stopCh)
	return nil
}

// WebSocketConfigWatcher implements ConfigWatcher for WebSocket notifications
type WebSocketConfigWatcher struct {
	url     string
	stopCh  chan struct{}
}

// NewWebSocketConfigWatcher creates a new WebSocket configuration watcher
func NewWebSocketConfigWatcher(url string) *WebSocketConfigWatcher {
	return &WebSocketConfigWatcher{
		url:    url,
		stopCh: make(chan struct{}),
	}
}

// Watch watches for WebSocket configuration updates
func (wscw *WebSocketConfigWatcher) Watch(ctx context.Context, callback func(*MLConfig) error) error {
	// This is a placeholder implementation
	// In a real implementation, you would:
	// 1. Connect to WebSocket
	// 2. Listen for configuration updates
	// 3. Parse and validate the configuration
	// 4. Call the callback with the new config
	
	ticker := time.NewTicker(30 * time.Second) // Placeholder ticker
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-wscw.stopCh:
			return fmt.Errorf("watcher stopped")
		case <-ticker.C:
			// Placeholder for WebSocket message handling
		}
	}
}

// Stop stops the WebSocket watcher
func (wscw *WebSocketConfigWatcher) Stop() error {
	close(wscw.stopCh)
	return nil
}

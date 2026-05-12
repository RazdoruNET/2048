package ml

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PersistenceManager handles ML configuration and data persistence
type PersistenceManager struct {
	mu              sync.RWMutex
	configPath      string
	dataPath        string
	backupPath      string
	autoSave        bool
	saveInterval    time.Duration
	maxBackups      int
	ctx             context.Context
	cancel          context.CancelFunc
	notifications   []NotificationCallback
}

// NotificationCallback is called when persistence events occur
type NotificationCallback func(event PersistenceEvent) error

// PersistenceEvent represents a persistence event
type PersistenceEvent struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
	Message   string      `json:"message"`
}

// PersistentData represents all persistent ML data
type PersistentData struct {
	Config       *MLConfig                `json:"config"`
	Techniques   map[string]float64       `json:"techniques"`
	Feedback     []FeedbackEntry          `json:"feedback"`
	Statistics   map[string]interface{}   `json:"statistics"`
	Version      string                   `json:"version"`
	LastModified time.Time               `json:"last_modified"`
}

// NewPersistenceManager creates a new persistence manager
func NewPersistenceManager(configPath, dataPath string) *PersistenceManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &PersistenceManager{
		configPath:    configPath,
		dataPath:      dataPath,
		backupPath:    filepath.Join(filepath.Dir(dataPath), "backups"),
		autoSave:      true,
		saveInterval:  5 * time.Minute,
		maxBackups:    10,
		ctx:           ctx,
		cancel:        cancel,
		notifications: make([]NotificationCallback, 0),
	}
}

// Start starts the persistence manager
func (pm *PersistenceManager) Start() error {
	// Create directories if they don't exist
	if err := pm.createDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %v", err)
	}
	
	// Start auto-save if enabled
	if pm.autoSave {
		go pm.autoSaveLoop()
	}
	
	// Send notification
	pm.sendNotification("started", "Persistence manager started")
	
	return nil
}

// Stop stops the persistence manager
func (pm *PersistenceManager) Stop() {
	pm.cancel()
	
	// Final save before stopping
	if pm.autoSave {
		pm.sendNotification("stopped", "Persistence manager stopped")
	}
}

// createDirectories creates necessary directories
func (pm *PersistenceManager) createDirectories() error {
	dirs := []string{
		filepath.Dir(pm.configPath),
		filepath.Dir(pm.dataPath),
		pm.backupPath,
	}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	
	return nil
}

// SaveConfig saves ML configuration to file
func (pm *PersistenceManager) SaveConfig(config *MLConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	// Create backup before saving
	if err := pm.createBackup(pm.configPath); err != nil {
		pm.sendNotification("backup_error", fmt.Sprintf("Failed to backup config: %v", err))
	}
	
	// Save configuration
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}
	
	if err := os.WriteFile(pm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}
	
	// Send notification
	pm.sendNotification("config_saved", "Configuration saved successfully")
	
	return nil
}

// LoadConfig loads ML configuration from file
func (pm *PersistenceManager) LoadConfig() (*MLConfig, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	data, err := os.ReadFile(pm.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return &MLConfig{
				Enabled:        true,
				UpdateInterval: time.Hour,
				MaxRetries:     3,
				Timeout:        30 * time.Second,
				CustomParameters: make(map[string]interface{}),
			}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}
	
	var config MLConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}
	
	// Send notification
	pm.sendNotification("config_loaded", "Configuration loaded successfully")
	
	return &config, nil
}

// SaveData saves all ML data to file
func (pm *PersistenceManager) SaveData(data *PersistentData) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	// Create backup before saving
	if err := pm.createBackup(pm.dataPath); err != nil {
		pm.sendNotification("backup_error", fmt.Sprintf("Failed to backup data: %v", err))
	}
	
	// Update metadata
	data.LastModified = time.Now()
	data.Version = "1.0"
	
	// Save data
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %v", err)
	}
	
	if err := os.WriteFile(pm.dataPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write data file: %v", err)
	}
	
	// Send notification
	pm.sendNotification("data_saved", "ML data saved successfully")
	
	return nil
}

// LoadData loads all ML data from file
func (pm *PersistenceManager) LoadData() (*PersistentData, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	data, err := os.ReadFile(pm.dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default data if file doesn't exist
			return &PersistentData{
				Config:     &MLConfig{Enabled: true},
				Techniques: make(map[string]float64),
				Feedback:   make([]FeedbackEntry, 0),
				Statistics: make(map[string]interface{}),
				Version:    "1.0",
			}, nil
		}
		return nil, fmt.Errorf("failed to read data file: %v", err)
	}
	
	var persistentData PersistentData
	if err := json.Unmarshal(data, &persistentData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %v", err)
	}
	
	// Send notification
	pm.sendNotification("data_loaded", "ML data loaded successfully")
	
	return &persistentData, nil
}

// createBackup creates a backup of a file
func (pm *PersistenceManager) createBackup(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // File doesn't exist, no backup needed
	}
	
	// Generate backup filename
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("%s_%s%s", filepath.Base(filePath), timestamp, filepath.Ext(filePath))
	backupPath := filepath.Join(pm.backupPath, backupName)
	
	// Copy file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return err
	}
	
	// Clean old backups
	go pm.cleanOldBackups()
	
	return nil
}

// cleanOldBackups removes old backup files
func (pm *PersistenceManager) cleanOldBackups() {
	files, err := filepath.Glob(filepath.Join(pm.backupPath, "*"))
	if err != nil {
		return
	}
	
	// Sort by modification time (newest first)
	type fileInfo struct {
		path    string
		modTime time.Time
	}
	
	var fileInfos []fileInfo
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		fileInfos = append(fileInfos, fileInfo{
			path:    file,
			modTime: info.ModTime(),
		})
	}
	
	// Keep only the most recent backups
	if len(fileInfos) > pm.maxBackups {
		for i := pm.maxBackups; i < len(fileInfos); i++ {
			os.Remove(fileInfos[i].path)
		}
	}
}

// autoSaveLoop runs the auto-save loop
func (pm *PersistenceManager) autoSaveLoop() {
	ticker := time.NewTicker(pm.saveInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			// Auto-save logic would be implemented here
			// This is a placeholder - in a real implementation,
			// you would save current ML engine state
		}
	}
}

// AddNotificationCallback adds a notification callback
func (pm *PersistenceManager) AddNotificationCallback(callback NotificationCallback) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	pm.notifications = append(pm.notifications, callback)
}

// RemoveNotificationCallback removes a notification callback
func (pm *PersistenceManager) RemoveNotificationCallback(callbackToRemove NotificationCallback) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	callbacks := make([]NotificationCallback, 0, len(pm.notifications))
	for _, callback := range pm.notifications {
		if fmt.Sprintf("%p", callback) != fmt.Sprintf("%p", callbackToRemove) {
			callbacks = append(callbacks, callback)
		}
	}
	pm.notifications = callbacks
}

// sendNotification sends a notification to all callbacks
func (pm *PersistenceManager) sendNotification(eventType, message string) {
	event := PersistenceEvent{
		Type:      eventType,
		Timestamp: time.Now(),
		Message:   message,
	}
	
	for _, callback := range pm.notifications {
		if err := callback(event); err != nil {
			// Log error but don't stop other notifications
			fmt.Printf("Notification callback error: %v\n", err)
		}
	}
}

// GetBackupList returns a list of available backups
func (pm *PersistenceManager) GetBackupList() ([]string, error) {
	files, err := filepath.Glob(filepath.Join(pm.backupPath, "*"))
	if err != nil {
		return nil, err
	}
	
	var backups []string
	for _, file := range files {
		backups = append(backups, filepath.Base(file))
	}
	
	return backups, nil
}

// RestoreFromBackup restores from a backup file
func (pm *PersistenceManager) RestoreFromBackup(backupName string) error {
	backupPath := filepath.Join(pm.backupPath, backupName)
	
	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backupName)
	}
	
	// Determine target file based on backup name
	var targetPath string
	if filepath.Base(backupName) == "config" {
		targetPath = pm.configPath
	} else {
		targetPath = pm.dataPath
	}
	
	// Read backup data
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %v", err)
	}
	
	// Create backup of current file before restoring
	if err := pm.createBackup(targetPath); err != nil {
		pm.sendNotification("backup_error", fmt.Sprintf("Failed to backup before restore: %v", err))
	}
	
	// Restore from backup
	if err := os.WriteFile(targetPath, data, 0644); err != nil {
		return fmt.Errorf("failed to restore backup: %v", err)
	}
	
	// Send notification
	pm.sendNotification("restored", fmt.Sprintf("Restored from backup: %s", backupName))
	
	return nil
}

// SetAutoSave enables or disables auto-save
func (pm *PersistenceManager) SetAutoSave(enabled bool, interval time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	pm.autoSave = enabled
	if interval > 0 {
		pm.saveInterval = interval
	}
}

// GetStatistics returns persistence statistics
func (pm *PersistenceManager) GetStatistics() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	stats := make(map[string]interface{})
	
	// Basic stats
	stats["auto_save_enabled"] = pm.autoSave
	stats["save_interval"] = pm.saveInterval.String()
	stats["max_backups"] = pm.maxBackups
	stats["config_path"] = pm.configPath
	stats["data_path"] = pm.dataPath
	stats["backup_path"] = pm.backupPath
	
	// File stats
	if configInfo, err := os.Stat(pm.configPath); err == nil {
		stats["config_size"] = configInfo.Size()
		stats["config_modified"] = configInfo.ModTime()
	}
	
	if dataInfo, err := os.Stat(pm.dataPath); err == nil {
		stats["data_size"] = dataInfo.Size()
		stats["data_modified"] = dataInfo.ModTime()
	}
	
	// Backup stats
	if backups, err := pm.GetBackupList(); err == nil {
		stats["backup_count"] = len(backups)
		stats["backups"] = backups
	}
	
	return stats
}

// WebSocketNotifier implements WebSocket notifications for persistence events
type WebSocketNotifier struct {
	clients map[string]chan PersistenceEvent
	mu      sync.RWMutex
}

// NewWebSocketNotifier creates a new WebSocket notifier
func NewWebSocketNotifier() *WebSocketNotifier {
	return &WebSocketNotifier{
		clients: make(map[string]chan PersistenceEvent),
	}
}

// Subscribe subscribes a client to persistence notifications
func (wsn *WebSocketNotifier) Subscribe(clientID string) chan PersistenceEvent {
	wsn.mu.Lock()
	defer wsn.mu.Unlock()
	
	ch := make(chan PersistenceEvent, 100) // Buffer 100 events
	wsn.clients[clientID] = ch
	
	return ch
}

// Unsubscribe unsubscribes a client from persistence notifications
func (wsn *WebSocketNotifier) Unsubscribe(clientID string) {
	wsn.mu.Lock()
	defer wsn.mu.Unlock()
	
	if ch, exists := wsn.clients[clientID]; exists {
		close(ch)
		delete(wsn.clients, clientID)
	}
}

// Notify sends a notification to all subscribed clients
func (wsn *WebSocketNotifier) Notify(event PersistenceEvent) {
	wsn.mu.RLock()
	defer wsn.mu.RUnlock()
	
	for clientID, ch := range wsn.clients {
		select {
		case ch <- event:
			// Event sent successfully
		default:
			// Channel is full, skip this client
			fmt.Printf("Notification channel full for client: %s\n", clientID)
		}
	}
}

// GetClientCount returns the number of subscribed clients
func (wsn *WebSocketNotifier) GetClientCount() int {
	wsn.mu.RLock()
	defer wsn.mu.RUnlock()
	
	return len(wsn.clients)
}

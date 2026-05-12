package ml

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// ModelManager manages ML model versions, deployment, and lifecycle
type ModelManager struct {
	mu              sync.RWMutex
	models          map[string]*ModelInfo
	activeModel     *ModelInfo
	modelPath       string
	backupPath      string
	maxVersions     int
	ctx             context.Context
	cancel          context.CancelFunc
	deployCallbacks []DeployCallback
}

// ModelInfo represents information about a model version
type ModelInfo struct {
	Version     string                 `json:"version"`
	Name        string                 `json:"name"`
	CreatedAt   time.Time              `json:"created_at"`
	Size        int64                  `json:"size"`
	Checksum    string                 `json:"checksum"`
	Performance *ModelPerformance      `json:"performance"`
	Metadata    map[string]interface{} `json:"metadata"`
	Status      ModelStatus            `json:"status"`
	DeployedAt  time.Time              `json:"deployed_at"`
	DeployedBy  string                 `json:"deployed_by"`
	FilePath    string                 `json:"file_path"`
	Description string                 `json:"description"`
	Tags        []string               `json:"tags"`
}

// ModelStatus represents the status of a model
type ModelStatus string

const (
	ModelStatusCreated    ModelStatus = "created"
	ModelStatusTraining   ModelStatus = "training"
	ModelStatusValidating ModelStatus = "validating"
	ModelStatusReady      ModelStatus = "ready"
	ModelStatusDeployed   ModelStatus = "deployed"
	ModelStatusDeprecated ModelStatus = "deprecated"
	ModelStatusArchived   ModelStatus = "archived"
	ModelStatusFailed     ModelStatus = "failed"
)

// DeployCallback is called when a model is deployed
type DeployCallback func(oldModel, newModel *ModelInfo) error

// NewModelManager creates a new model manager
func NewModelManager(modelPath string, maxVersions int) *ModelManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &ModelManager{
		models:          make(map[string]*ModelInfo),
		modelPath:       modelPath,
		backupPath:      filepath.Join(filepath.Dir(modelPath), "model_backups"),
		maxVersions:     maxVersions,
		ctx:             ctx,
		cancel:          cancel,
		deployCallbacks: make([]DeployCallback, 0),
	}
}

// Start starts the model manager
func (mm *ModelManager) Start() error {
	// Create directories
	if err := os.MkdirAll(mm.modelPath, 0755); err != nil {
		return fmt.Errorf("failed to create model directory: %v", err)
	}

	if err := os.MkdirAll(mm.backupPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %v", err)
	}

	// Load existing models
	if err := mm.loadExistingModels(); err != nil {
		log.Printf("Warning: Failed to load existing models: %v", err)
	}

	// Find active model
	if err := mm.findActiveModel(); err != nil {
		log.Printf("Warning: Failed to find active model: %v", err)
	}

	log.Printf("Model manager started with %d models", len(mm.models))
	return nil
}

// Stop stops the model manager
func (mm *ModelManager) Stop() {
	mm.cancel()
	log.Println("Model manager stopped")
}

// RegisterModel registers a new model version
func (mm *ModelManager) RegisterModel(version, name, description string, performance *ModelPerformance, metadata map[string]interface{}, tags []string) (*ModelInfo, error) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	// Check if model version already exists
	if _, exists := mm.models[version]; exists {
		return nil, fmt.Errorf("model version %s already exists", version)
	}

	// Create model info
	model := &ModelInfo{
		Version:     version,
		Name:        name,
		CreatedAt:   time.Now(),
		Performance: performance,
		Metadata:    metadata,
		Status:      ModelStatusCreated,
		Description: description,
		Tags:        tags,
		FilePath:    filepath.Join(mm.modelPath, fmt.Sprintf("model_%s.bin", version)),
	}

	// Add to models
	mm.models[version] = model

	// Clean up old versions if needed
	if len(mm.models) > mm.maxVersions {
		mm.cleanupOldVersions()
	}

	log.Printf("Registered model version %s", version)
	return model, nil
}

// DeployModel deploys a model version
func (mm *ModelManager) DeployModel(version, deployedBy string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	model, exists := mm.models[version]
	if !exists {
		return fmt.Errorf("model version %s not found", version)
	}

	// Check if model is ready for deployment
	if model.Status != ModelStatusReady {
		return fmt.Errorf("model %s is not ready for deployment (status: %s)", version, model.Status)
	}

	// Store old model for callback
	oldModel := mm.activeModel

	// Update model status
	model.Status = ModelStatusDeployed
	model.DeployedAt = time.Now()
	model.DeployedBy = deployedBy

	// Update active model
	mm.activeModel = model

	// Call deploy callbacks
	for _, callback := range mm.deployCallbacks {
		if err := callback(oldModel, model); err != nil {
			log.Printf("Deploy callback error: %v", err)
		}
	}

	// Update ML engine
	if err := mm.updateMLEngine(model); err != nil {
		// Rollback on failure
		mm.activeModel = oldModel
		if oldModel != nil {
			oldModel.Status = ModelStatusDeployed
		}
		model.Status = ModelStatusReady
		return fmt.Errorf("failed to update ML engine: %v", err)
	}

	// Deprecate old model
	if oldModel != nil {
		oldModel.Status = ModelStatusDeprecated
	}

	log.Printf("Deployed model version %s", version)
	return nil
}

// GetModel returns model information by version
func (mm *ModelManager) GetModel(version string) (*ModelInfo, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	model, exists := mm.models[version]
	if !exists {
		return nil, fmt.Errorf("model version %s not found", version)
	}

	// Return a copy
	modelCopy := *model
	return &modelCopy, nil
}

// GetActiveModel returns the currently active model
func (mm *ModelManager) GetActiveModel() *ModelInfo {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	if mm.activeModel == nil {
		return nil
	}

	// Return a copy
	modelCopy := *mm.activeModel
	return &modelCopy
}

// ListModels returns all models
func (mm *ModelManager) ListModels() []*ModelInfo {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	models := make([]*ModelInfo, 0, len(mm.models))
	for _, model := range mm.models {
		modelCopy := *model
		models = append(models, &modelCopy)
	}

	// Sort by creation time (newest first)
	sort.Slice(models, func(i, j int) bool {
		return models[i].CreatedAt.After(models[j].CreatedAt)
	})

	return models
}

// UpdateModelStatus updates the status of a model
func (mm *ModelManager) UpdateModelStatus(version string, status ModelStatus) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	model, exists := mm.models[version]
	if !exists {
		return fmt.Errorf("model version %s not found", version)
	}

	model.Status = status
	log.Printf("Updated model %s status to %s", version, status)

	return nil
}

// UpdateModelPerformance updates the performance metrics of a model
func (mm *ModelManager) UpdateModelPerformance(version string, performance *ModelPerformance) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	model, exists := mm.models[version]
	if !exists {
		return fmt.Errorf("model version %s not found", version)
	}

	model.Performance = performance
	log.Printf("Updated model %s performance metrics", version)

	return nil
}

// DeleteModel deletes a model version
func (mm *ModelManager) DeleteModel(version string, force bool) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	model, exists := mm.models[version]
	if !exists {
		return fmt.Errorf("model version %s not found", version)
	}

	// Cannot delete active model unless forced
	if mm.activeModel != nil && mm.activeModel.Version == version && !force {
		return fmt.Errorf("cannot delete active model %s (use force=true)", version)
	}

	// Create backup before deletion
	if err := mm.createModelBackup(model); err != nil {
		log.Printf("Warning: Failed to create backup before deletion: %v", err)
	}

	// Remove model file
	if _, err := os.Stat(model.FilePath); err == nil {
		if err := os.Remove(model.FilePath); err != nil {
			log.Printf("Warning: Failed to remove model file: %v", err)
		}
	}

	// Remove from models
	delete(mm.models, version)

	// Clear active model if it was the deleted one
	if mm.activeModel != nil && mm.activeModel.Version == version {
		mm.activeModel = nil
	}

	log.Printf("Deleted model version %s", version)
	return nil
}

// AddDeployCallback adds a deploy callback
func (mm *ModelManager) AddDeployCallback(callback DeployCallback) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.deployCallbacks = append(mm.deployCallbacks, callback)
}

// RemoveDeployCallback removes a deploy callback
func (mm *ModelManager) RemoveDeployCallback(callbackToRemove DeployCallback) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	callbacks := make([]DeployCallback, 0, len(mm.deployCallbacks))
	for _, callback := range mm.deployCallbacks {
		if fmt.Sprintf("%p", callback) != fmt.Sprintf("%p", callbackToRemove) {
			callbacks = append(callbacks, callback)
		}
	}
	mm.deployCallbacks = callbacks
}

// GetStatistics returns model management statistics
func (mm *ModelManager) GetStatistics() map[string]interface{} {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	stats := make(map[string]interface{})

	stats["total_models"] = len(mm.models)
	stats["max_versions"] = mm.maxVersions
	stats["model_path"] = mm.modelPath
	stats["backup_path"] = mm.backupPath

	// Count by status
	statusCounts := make(map[ModelStatus]int)
	for _, model := range mm.models {
		statusCounts[model.Status]++
	}

	stats["status_counts"] = statusCounts

	// Active model info
	if mm.activeModel != nil {
		stats["active_model"] = map[string]interface{}{
			"version":     mm.activeModel.Version,
			"name":        mm.activeModel.Name,
			"deployed_at": mm.activeModel.DeployedAt,
			"deployed_by": mm.activeModel.DeployedBy,
		}
	} else {
		stats["active_model"] = nil
	}

	// Model sizes
	var totalSize int64
	for _, model := range mm.models {
		totalSize += model.Size
	}
	stats["total_size_bytes"] = totalSize
	stats["average_size_bytes"] = int64(0)
	if len(mm.models) > 0 {
		stats["average_size_bytes"] = totalSize / int64(len(mm.models))
	}

	return stats
}

// loadExistingModels loads existing models from disk
func (mm *ModelManager) loadExistingModels() error {
	// This would scan the model directory and load model metadata
	// For now, create a default model if none exist

	if len(mm.models) == 0 {
		defaultModel := &ModelInfo{
			Version:   "1.0.0",
			Name:      "Default DPI Model",
			CreatedAt: time.Now(),
			Size:      1024 * 1024, // 1MB mock size
			Checksum:  "default_checksum",
			Performance: &ModelPerformance{
				Accuracy:       0.85,
				Precision:      0.83,
				Recall:         0.87,
				F1Score:        0.85,
				Loss:           0.25,
				ValidationLoss: 0.27,
				TestLoss:       0.28,
				TrainingTime:   10 * time.Minute,
			},
			Metadata: map[string]interface{}{
				"training_data_size": 10000,
				"training_time":      "10m",
				"algorithm":          "neural_network",
			},
			Status:      ModelStatusReady,
			Description: "Default DPI detection model",
			Tags:        []string{"default", "dpi", "baseline"},
			FilePath:    filepath.Join(mm.modelPath, "model_1.0.0.bin"),
		}

		mm.models["1.0.0"] = defaultModel
	}

	return nil
}

// findActiveModel finds and sets the active model
func (mm *ModelManager) findActiveModel() error {
	// Look for the most recently deployed model
	var latestModel *ModelInfo
	var latestTime time.Time

	for _, model := range mm.models {
		if model.Status == ModelStatusDeployed && model.DeployedAt.After(latestTime) {
			latestModel = model
			latestTime = model.DeployedAt
		}
	}

	if latestModel == nil {
		// If no deployed model, use the latest ready model
		for _, model := range mm.models {
			if model.Status == ModelStatusReady && model.CreatedAt.After(latestTime) {
				latestModel = model
				latestTime = model.CreatedAt
			}
		}
	}

	mm.activeModel = latestModel
	return nil
}

// cleanupOldVersions removes old model versions
func (mm *ModelManager) cleanupOldVersions() {
	// Sort models by creation time
	type modelSort struct {
		version   string
		createdAt time.Time
	}

	var sortedModels []modelSort
	for version, model := range mm.models {
		sortedModels = append(sortedModels, modelSort{
			version:   version,
			createdAt: model.CreatedAt,
		})
	}

	sort.Slice(sortedModels, func(i, j int) bool {
		return sortedModels[i].createdAt.After(sortedModels[j].createdAt)
	})

	// Keep only the newest maxVersions models
	if len(sortedModels) > mm.maxVersions {
		for i := mm.maxVersions; i < len(sortedModels); i++ {
			version := sortedModels[i].version
			model := mm.models[version]

			// Don't delete active model
			if mm.activeModel != nil && mm.activeModel.Version == version {
				continue
			}

			// Create backup before deletion
			if err := mm.createModelBackup(model); err != nil {
				log.Printf("Warning: Failed to create backup for old model %s: %v", version, err)
			}

			// Remove model file
			if _, err := os.Stat(model.FilePath); err == nil {
				os.Remove(model.FilePath)
			}

			// Remove from models
			delete(mm.models, version)

			log.Printf("Cleaned up old model version %s", version)
		}
	}
}

// createModelBackup creates a backup of a model
func (mm *ModelManager) createModelBackup(model *ModelInfo) error {
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("model_%s_%s_backup.bin", model.Version, timestamp)
	backupPath := filepath.Join(mm.backupPath, backupName)

	// Copy model file to backup location
	if _, err := os.Stat(model.FilePath); err == nil {
		data, err := os.ReadFile(model.FilePath)
		if err != nil {
			return err
		}

		return os.WriteFile(backupPath, data, 0644)
	}

	return nil
}

// updateMLEngine updates the ML engine with the new model
func (mm *ModelManager) updateMLEngine(model *ModelInfo) error {
	// This would load the model file and update the ML engine
	// For now, just update the status
	log.Printf("Updated ML engine with model %s", model.Version)
	return nil
}

// RollbackToModel rolls back to a previous model version
func (mm *ModelManager) RollbackToModel(version, deployedBy string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	model, exists := mm.models[version]
	if !exists {
		return fmt.Errorf("model version %s not found", version)
	}

	// Store current active model
	currentActive := mm.activeModel

	// Deploy the rollback model
	model.Status = ModelStatusDeployed
	model.DeployedAt = time.Now()
	model.DeployedBy = deployedBy
	mm.activeModel = model

	// Mark previous model as deprecated
	if currentActive != nil {
		currentActive.Status = ModelStatusDeprecated
	}

	// Update ML engine
	if err := mm.updateMLEngine(model); err != nil {
		// Rollback failed, restore previous state
		mm.activeModel = currentActive
		if currentActive != nil {
			currentActive.Status = ModelStatusDeployed
		}
		model.Status = ModelStatusReady
		return fmt.Errorf("rollback failed: %v", err)
	}

	log.Printf("Rolled back to model version %s", version)
	return nil
}

// CompareModels compares two models
func (mm *ModelManager) CompareModels(version1, version2 string) (map[string]interface{}, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	model1, exists1 := mm.models[version1]
	model2, exists2 := mm.models[version2]

	if !exists1 {
		return nil, fmt.Errorf("model version %s not found", version1)
	}
	if !exists2 {
		return nil, fmt.Errorf("model version %s not found", version2)
	}

	comparison := make(map[string]interface{})

	// Basic comparison
	comparison["version1"] = map[string]interface{}{
		"version": model1.Version,
		"name":    model1.Name,
		"status":  model1.Status,
		"size":    model1.Size,
		"created": model1.CreatedAt,
	}

	comparison["version2"] = map[string]interface{}{
		"version": model2.Version,
		"name":    model2.Name,
		"status":  model2.Status,
		"size":    model2.Size,
		"created": model2.CreatedAt,
	}

	// Performance comparison
	if model1.Performance != nil && model2.Performance != nil {
		comparison["performance_diff"] = map[string]interface{}{
			"accuracy_diff":  model2.Performance.Accuracy - model1.Performance.Accuracy,
			"precision_diff": model2.Performance.Precision - model1.Performance.Precision,
			"recall_diff":    model2.Performance.Recall - model1.Performance.Recall,
			"f1_score_diff":  model2.Performance.F1Score - model1.Performance.F1Score,
			"loss_diff":      model1.Performance.Loss - model2.Performance.Loss,
		}
	}

	// Size comparison
	comparison["size_diff"] = model2.Size - model1.Size
	comparison["size_diff_percent"] = float64(model2.Size-model1.Size) / float64(model1.Size) * 100

	// Time difference
	comparison["time_diff"] = model2.CreatedAt.Sub(model1.CreatedAt)

	return comparison, nil
}

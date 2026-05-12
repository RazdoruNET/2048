package handlers

import (
	"encoding/json"
	"math/rand"
	"time"

	"socks5-dpi-proxy/internal/storage"
)

type DemoDataGenerator struct {
	storage *storage.Storage
	wsHandler *WebSocketHandler
}

func NewDemoDataGenerator(storage *storage.Storage, wsHandler *WebSocketHandler) *DemoDataGenerator {
	return &DemoDataGenerator{
		storage:   storage,
		wsHandler: wsHandler,
	}
}

func (d *DemoDataGenerator) GenerateDemoData() error {
	// Generate demo requests
	domains := []string{"google.com", "youtube.com", "facebook.com", "twitter.com", "instagram.com", "github.com", "stackoverflow.com", "reddit.com"}
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	dpiTypes := []string{"Signature", "Behavioral", "MLBased", "Hybrid"}
	techniques := [][]string{
		{"fragmentation", "headers"},
		{"encryption", "protocol_mask"},
		{"fragmentation", "timing"},
		{"headers", "encryption"},
		{"protocol_mask", "timing"},
	}

	// Generate 50 demo requests
	for i := 0; i < 50; i++ {
		domain := domains[rand.Intn(len(domains))]
		method := methods[rand.Intn(len(methods))]
		dpiType := dpiTypes[rand.Intn(len(dpiTypes))]
		techniqueList := techniques[rand.Intn(len(techniques))]
		
		success := rand.Float32() > 0.2 // 80% success rate
		latency := int64(rand.Intn(500) + 50) // 50-550ms
		bytesIn := int64(rand.Intn(10000) + 1000)
		bytesOut := int64(rand.Intn(5000) + 500)
		confidence := rand.Float64()*0.4 + 0.6 // 0.6-1.0

		// Create techniques JSON
		techniquesJSON, _ := json.Marshal(techniqueList)

		request := &storage.RequestLog{
			RequestID:  generateRequestID(),
			Domain:     domain,
			Method:     method,
			StatusCode: 200,
			Success:    success,
			Latency:    latency,
			BytesIn:    bytesIn,
			BytesOut:   bytesOut,
			Techniques: string(techniquesJSON),
			DPIType:    dpiType,
			Confidence: confidence,
			CreatedAt:  time.Now().Add(-time.Duration(i) * time.Minute),
			UpdatedAt:  time.Now().Add(-time.Duration(i) * time.Minute),
		}

		if err := d.storage.LogRequest(request); err != nil {
			return err
		}

		// Broadcast via WebSocket
		if d.wsHandler != nil {
			d.wsHandler.BroadcastRequestCompleted(RequestCompleted{
				RequestID:   request.RequestID,
				Domain:      request.Domain,
				Success:     request.Success,
				Latency:     request.Latency,
				Techniques:  techniqueList,
				DPIType:     request.DPIType,
				Confidence:  request.Confidence,
				CompletedAt: request.CreatedAt,
			})
		}
	}

	// Generate technique effectiveness data
	for _, domain := range domains {
		for _, technique := range []string{"fragmentation", "headers", "encryption", "protocol_mask", "timing"} {
			effectiveness := rand.Float64()*0.6 + 0.4 // 0.4-1.0
			sampleCount := rand.Intn(100) + 10

			techEffectiveness := &storage.TechniqueEffectiveness{
				Technique:     technique,
				Domain:        domain,
				Effectiveness: effectiveness,
				SampleCount:   sampleCount,
				LastUpdate:    time.Now().Add(-time.Duration(rand.Intn(60)) * time.Minute),
			}

			if err := d.storage.UpdateTechniqueEffectiveness(techEffectiveness); err != nil {
				return err
			}

			// Broadcast via WebSocket
			if d.wsHandler != nil {
				d.wsHandler.BroadcastTechniqueEffectiveness(TechniqueEffectivenessChanged{
					Technique:     technique,
					Domain:        domain,
					Effectiveness: effectiveness,
					SampleCount:   sampleCount,
					UpdatedAt:     techEffectiveness.LastUpdate,
				})
			}
		}
	}

	// Generate ML statistics
	stats := &storage.MLStatistics{
		Timestamp:         time.Now(),
		TotalRequests:     50,
		SuccessRate:       0.8,
		AverageLatency:    250,
		ThroughputBPS:     1024000,
		ErrorRate:         0.2,
		ActiveConnections: 5,
	}

	if err := d.storage.SaveMLStatistics(stats); err != nil {
		return err
	}

	return nil
}

func (d *DemoDataGenerator) StartContinuousDemo() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// Generate a new request every 5 seconds
			domains := []string{"google.com", "youtube.com", "facebook.com", "twitter.com", "instagram.com"}
			methods := []string{"GET", "POST", "PUT", "DELETE"}
			dpiTypes := []string{"Signature", "Behavioral", "MLBased", "Hybrid"}
			techniques := [][]string{
				{"fragmentation", "headers"},
				{"encryption", "protocol_mask"},
				{"fragmentation", "timing"},
				{"headers", "encryption"},
				{"protocol_mask", "timing"},
			}

			domain := domains[rand.Intn(len(domains))]
			method := methods[rand.Intn(len(methods))]
			dpiType := dpiTypes[rand.Intn(len(dpiTypes))]
			techniqueList := techniques[rand.Intn(len(techniques))]
			
			success := rand.Float32() > 0.2 // 80% success rate
			latency := int64(rand.Intn(500) + 50) // 50-550ms
			bytesIn := int64(rand.Intn(10000) + 1000)
			bytesOut := int64(rand.Intn(5000) + 500)
			confidence := rand.Float64()*0.4 + 0.6 // 0.6-1.0

			// Create techniques JSON
			techniquesJSON, _ := json.Marshal(techniqueList)

			request := &storage.RequestLog{
				RequestID:  generateRequestID(),
				Domain:     domain,
				Method:     method,
				StatusCode: 200,
				Success:    success,
				Latency:    latency,
				BytesIn:    bytesIn,
				BytesOut:   bytesOut,
				Techniques: string(techniquesJSON),
				DPIType:    dpiType,
				Confidence: confidence,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}

			d.storage.LogRequest(request)

			// Broadcast via WebSocket
			if d.wsHandler != nil {
				d.wsHandler.BroadcastRequestCompleted(RequestCompleted{
					RequestID:   request.RequestID,
					Domain:      request.Domain,
					Success:     request.Success,
					Latency:     request.Latency,
					Techniques:  techniqueList,
					DPIType:     request.DPIType,
					Confidence:  request.Confidence,
					CompletedAt: request.CreatedAt,
				})
			}
		}
	}()
}

func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randString(8)
}

func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

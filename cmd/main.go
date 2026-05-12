package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"socks5-dpi-proxy/internal/final"
)

func main() {
	// Create final configuration
	config := &final.FinalConfig{
		Core: &final.CoreConfig{
			ListenAddress:  "0.0.0.0",
			ListenPort:     1080,
			MaxConnections: 10000,
			Timeout:        30 * time.Second,
			EnableLogging:  true,
			LogLevel:       "info",
		},
		Integration: &final.IntegrationConfig{
			AutoStart:                  true,
			HealthCheckInterval:        30 * time.Second,
			MetricsAggregation:         true,
			CrossComponentOptimization: true,
			EnableFailover:             true,
			EnableLoadBalancing:        true,
			EnableAutoScaling:          false,
		},
	}

	// Create final manager
	finalManager := final.NewFinalManager(config)

	// Create test manager
	testConfig := &final.TestConfig{
		EnabledSuites:   []string{"unit", "integration"},
		Timeout:         10 * time.Minute,
		ParallelTests:   2,
		Retries:         1,
		GenerateReports: true,
		OutputPath:      "./test-results",
	}
	testManager := final.NewTestManager(finalManager, testConfig)

	// Create deployment manager
	deployConfig := &final.DeploymentConfig{
		Environment: "production",
		Target: &final.TargetConfig{
			Hosts:       []string{"localhost"},
			Username:    "deploy",
			DeployPath:  "/opt/socks5-dpi-proxy",
			ServiceName: "socks5-dpi-proxy",
			Port:        1080,
		},
		HealthCheck: &final.HealthCheckConfig{
			Enabled:  true,
			Interval: 30 * time.Second,
			Timeout:  10 * time.Second,
			Retries:  3,
			Endpoints: []string{
				"http://localhost:8080/health",
				"http://localhost:8085/metrics",
			},
		},
		Rollback: &final.RollbackConfig{
			Enabled:        true,
			AutoRollback:   true,
			Threshold:      0.1,
			Window:         5 * time.Minute,
			BackupVersions: 3,
		},
		Monitoring: &final.MonitoringConfig{
			Enabled:         true,
			MetricsPort:     9090,
			LogLevel:        "info",
			AlertingEnabled: true,
		},
		Backup: &final.BackupConfig{
			Enabled:   true,
			Path:      "/opt/backups/socks5-dpi-proxy",
			Retention: 30 * 24 * time.Hour,
			Compress:  true,
		},
	}
	deploymentManager := final.NewDeploymentManager(finalManager, testManager, deployConfig)

	// Create documentation manager
	docConfig := &final.DocumentationConfig{
		OutputPath:       "./docs",
		Formats:          []string{"json", "markdown"},
		IncludeAPI:       true,
		IncludeConfig:    true,
		IncludeMetrics:   true,
		IncludeTests:     true,
		GenerateDiagrams: true,
	}
	docManager := final.NewDocumentationManager(finalManager, testManager, deploymentManager, docConfig)

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start the system
	log.Println("Starting SOCKS5 DPI Proxy System...")

	if err := finalManager.Start(ctx); err != nil {
		log.Fatalf("Failed to start final manager: %v", err)
	}

	// Generate documentation
	log.Println("Generating system documentation...")
	if err := docManager.GenerateDocumentation(); err != nil {
		log.Printf("Warning: Failed to generate documentation: %v", err)
	}

	// Run tests
	log.Println("Running system tests...")
	if err := testManager.RunTests(ctx); err != nil {
		log.Printf("Warning: Tests failed: %v", err)
	} else {
		log.Println("All tests passed!")
	}

	// Print system status
	printSystemStatus(finalManager)

	// Main loop
	log.Println("SOCKS5 DPI Proxy System is running...")
	log.Printf("Listening on %s:%d", config.Core.ListenAddress, config.Core.ListenPort)
	log.Println("Press Ctrl+C to stop")

	// Wait for shutdown signal
	<-sigCh
	log.Println("Shutdown signal received...")

	// Graceful shutdown
	log.Println("Stopping system...")
	if err := finalManager.Stop(); err != nil {
		log.Printf("Error stopping system: %v", err)
	}

	log.Println("System stopped successfully")
}

// printSystemStatus prints current system status
func printSystemStatus(manager *final.FinalManager) {
	fmt.Println("\n=== SOCKS5 DPI Proxy System Status ===")

	// Check if running
	if manager.IsRunning() {
		fmt.Println("✅ Status: Running")
	} else {
		fmt.Println("❌ Status: Not Running")
	}

	// Get health status
	health := manager.GetHealth()
	fmt.Printf("📊 Component Health: %d/%d healthy\n", countHealthy(health), len(health))

	for component, healthy := range health {
		status := "❌"
		if healthy {
			status = "✅"
		}
		fmt.Printf("  %s %s\n", status, component)
	}

	// Get metrics
	metrics := manager.GetMetrics()
	if metrics != nil {
		if metrics.Connections != nil {
			fmt.Printf("🔗 Connections: %d active, %d total\n",
				metrics.Connections.ActiveConnections,
				metrics.Connections.TotalConnections)
		}
		if metrics.System != nil {
			fmt.Printf("📈 System Resources: CPU %.1f%%, Memory %.1f%%\n",
				metrics.System.CPUUsage,
				metrics.System.MemoryUsage)
		}
	}

	fmt.Println("=====================================\n")
}

// countHealthy counts healthy components
func countHealthy(health map[string]bool) int {
	count := 0
	for _, healthy := range health {
		if healthy {
			count++
		}
	}
	return count
}

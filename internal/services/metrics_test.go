package services

import (
	"context"
	"testing"
	"time"

	"github.com/deezer/groroti/internal/config"
	"github.com/deezer/groroti/internal/middlewares"
	"github.com/deezer/groroti/internal/model"
)

// TestMetricsWithPrometheus tests that metrics work with Prometheus (legacy mode)
func TestMetricsWithPrometheus(t *testing.T) {
	// Initialize with Prometheus mode (OTel disabled)
	currentConfig = config.Config{
		EnableOTelMetrics: false,
	}

	// Initialize database for testing
	db := model.InitDatabase()
	defer db.Close()

	// Start recording metrics
	recordMetrics()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Create a test ROTI
	model.CreateROTI("Test ROTI", false, false, 30)

	// Wait for metrics to update
	time.Sleep(200 * time.Millisecond)

	// Test that we can get Prometheus metrics handler without panic
	handler := NewMetricsHandler()
	if handler == nil {
		t.Fatal("Expected non-nil Prometheus handler")
	}
}

// TestMetricsWithOTel tests that metrics work with OpenTelemetry
func TestMetricsWithOTel(t *testing.T) {
	// Initialize with OTel mode
	currentConfig = config.Config{
		EnableOTelMetrics: true,
		OTLPEndpoint:      "http://localhost:4318",
	}

	// Set up OTel metrics (it will fail to connect to endpoint but that's ok for this test)
	ctx := context.Background()
	shutdown, err := middlewares.SetupOTelMetrics(ctx, currentConfig)
	if err != nil {
		t.Logf("Expected error connecting to OTLP endpoint (this is ok for test): %v", err)
	}
	if shutdown != nil {
		defer shutdown(context.Background())
	}

	// Initialize database for testing
	db := model.InitDatabase()
	defer db.Close()

	// Start recording metrics
	recordMetrics()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Create a test ROTI
	model.CreateROTI("Test ROTI OTel", false, false, 30)

	// Wait for metrics to initialize and record
	time.Sleep(200 * time.Millisecond)

	// Verify that OTel metrics instruments were created
	if otelTotalRotis == nil {
		t.Error("Expected otelTotalRotis to be initialized")
	}
	if otelActiveRotis == nil {
		t.Error("Expected otelActiveRotis to be initialized")
	}
}

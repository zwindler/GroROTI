package services

import (
	"context"
	"net/http"
	"time"

	"github.com/deezer/groroti/internal/middlewares"
	"github.com/deezer/groroti/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/metric"
)

var (
	total_rotis = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "groroti_total_rotis",
		Help: "All the ROTIs that have been created since the beginning (including deleted ones)",
	})
	active_rotis = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "groroti_active_rotis",
		Help: "All the ROTIs that have been created and not deleted",
	})

	// OpenTelemetry metrics
	otelTotalRotis  metric.Int64Gauge
	otelActiveRotis metric.Int64Gauge
)

// NewMetricsHandler creates the handler allowing to dump Prometheus metrics
func NewMetricsHandler() http.Handler {
	prometheus.MustRegister(total_rotis)
	prometheus.MustRegister(active_rotis)

	return promhttp.Handler()
}

// initOTelMetrics initializes OpenTelemetry metrics instruments
func initOTelMetrics() error {
	if middlewares.MP == nil {
		return nil
	}

	meter := middlewares.MP.Meter("github.com/deezer/groroti/internal/services")

	var err error
	otelTotalRotis, err = meter.Int64Gauge("groroti.total.rotis",
		metric.WithDescription("All the ROTIs that have been created since the beginning (including deleted ones)"),
	)
	if err != nil {
		return err
	}

	otelActiveRotis, err = meter.Int64Gauge("groroti.active.rotis",
		metric.WithDescription("All the ROTIs that have been created and not deleted"),
	)
	if err != nil {
		return err
	}

	return nil
}

// create a goroutine that will periodically query the DB
// this allows to avoid querying too much the db
func recordMetrics() {
	// Initialize OTel metrics if enabled
	if currentConfig.EnableOTelMetrics {
		if err := initOTelMetrics(); err != nil {
			log.Error().Err(err).Msg("failed to initialize OpenTelemetry metrics")
		}
	}

	go func() {
		for {
			totalCount := int64(model.GetMaxROTIID())
			activeCount := int64(model.CountROTIs())

			// Record to Prometheus if OTel metrics are not enabled
			if !currentConfig.EnableOTelMetrics {
				total_rotis.Set(float64(totalCount))
				active_rotis.Set(float64(activeCount))
			}

			// Record to OpenTelemetry if enabled
			if currentConfig.EnableOTelMetrics && otelTotalRotis != nil && otelActiveRotis != nil {
				ctx := context.Background()
				otelTotalRotis.Record(ctx, totalCount)
				otelActiveRotis.Record(ctx, activeCount)
			}

			time.Sleep(15 * time.Second)
		}
	}()
}

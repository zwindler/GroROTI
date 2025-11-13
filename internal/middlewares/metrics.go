package middlewares

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/deezer/groroti/internal/config"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

var MP *sdkmetric.MeterProvider

// SetupOTelMetrics initializes OpenTelemetry with the OTLP exporter for metrics.
func SetupOTelMetrics(ctx context.Context, config config.Config) (func(context.Context) error, error) {
	log.Info().Msgf("enable OpenTelemetry metrics: %t", config.EnableOTelMetrics)
	if config.EnableOTelMetrics {
		// Determine if the endpoint uses HTTPS and configure appropriately
		var clientOptions []otlpmetrichttp.Option

		// Set Basic Authentication headers if credentials are provided
		if config.OTLPBasicUsername != "" && config.OTLPBasicPassword != "" {
			basicAuthHeader := generateBasicAuthHeader(config.OTLPBasicUsername, config.OTLPBasicPassword)
			clientOptions = append(clientOptions, otlpmetrichttp.WithHeaders(map[string]string{
				"Authorization": basicAuthHeader,
			}))
		}

		if strings.HasPrefix(config.OTLPEndpoint, "https://") {
			clientOptions = append(clientOptions,
				otlpmetrichttp.WithEndpoint(strings.TrimPrefix(config.OTLPEndpoint, "https://")),
				otlpmetrichttp.WithTLSClientConfig(&tls.Config{InsecureSkipVerify: false}),
			)
		} else {
			clientOptions = append(clientOptions,
				otlpmetrichttp.WithEndpoint(strings.TrimPrefix(config.OTLPEndpoint, "http://")),
				otlpmetrichttp.WithInsecure(),
			)
		}

		exporter, err := otlpmetrichttp.New(ctx, clientOptions...)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
		}

		res, err := resource.New(ctx,
			resource.WithAttributes(
				semconv.ServiceNameKey.String("GroROTI"),
			),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create resource: %w", err)
		}

		// Set up the MeterProvider with the exporter and resource
		MP = sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
			sdkmetric.WithResource(res),
		)

		// Set the global MeterProvider
		otel.SetMeterProvider(MP)

		// Function to shutdown the meter provider
		shutdown := func(ctx context.Context) error {
			// Ensure all metrics are exported before shutting down
			err := MP.Shutdown(ctx)
			if err != nil {
				log.Printf("failed to shutdown meter provider: %v", err)
			}
			return err
		}

		return shutdown, nil
	}
	return nil, nil
}

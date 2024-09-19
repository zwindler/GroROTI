package middlewares

import (
	"context"

	"github.com/deezer/groroti/internal/config"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func newExporter(ctx context.Context, endpoint string)  (*otlptrace.Exporter, error) {
		// Create a new OTLP HTTP exporter
		log.Info().Msgf("sending OpenTelemetry traces to: %s", endpoint)
		client := otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(endpoint),
			otlptracehttp.WithInsecure(), // TODO allow secure connections
		)
		exporter, err := otlptrace.New(ctx, client)
		if err != nil {
			return nil, err
		}
		return exporter, nil
}

func newTraceProvider(exp sdktrace.SpanExporter) *sdktrace.TracerProvider {
	// Ensure default SDK resources and the required service name are set.
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("GroROTI"),
		),
	)

	if err != nil {
		panic(err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	)
}

// SetupOTelSDK initializes OpenTelemetry with the OTLP exporter for tracing.
func SetupOTelSDK(ctx context.Context, config config.Config) (func(context.Context) error, *sdktrace.TracerProvider, error) {
	log.Info().Msgf("enable OpenTelemetry tracing: %t", config.EnableTracing)
	if config.EnableTracing {

		exp, err := newExporter(ctx, config.OTLPEndpoint)
		if err != nil {
			log.Error().Msgf ("failed to initialize OTEL exporter: %v", err)
			return nil, nil, err
		}
	
		// Create a new tracer provider with a batch span processor and the given exporter.
		tp := newTraceProvider(exp)
	
		// Handle shutdown properly so nothing leaks.
		defer func() { _ = tp.Shutdown(ctx) }()
	
		otel.SetTracerProvider(tp)

		// Function to shutdown the tracer provider
		shutdown := func(ctx context.Context) error {
			// Ensure all spans are exported before shutting down
			err := tp.Shutdown(ctx)
			if err != nil {
				log.Printf("failed to shutdown tracer provider: %v", err)
			}
			return err
		}

		return shutdown, tp, nil
	}
	return nil, nil, nil
}


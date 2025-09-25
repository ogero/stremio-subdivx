package common

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	metric2 "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.37.0"
)

// InitInstrumentation setups otel
func InitInstrumentation(serviceName, serviceVersion, serviceEnvironment, exporterEndpoint string) (func(ctx context.Context), error) {

	res, err := resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
			semconv.DeploymentEnvironmentName(serviceEnvironment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to merge otel resource: %w", err)
	}

	// Metric exporter
	metricExporter, err := otlpmetricgrpc.New(
		context.Background(),
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(exporterEndpoint),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	// Metric periodic reader
	metricPeriodicReader := metric.NewPeriodicReader(metricExporter, metric.WithInterval(30*time.Second))

	// Metric provider
	metricsProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metricPeriodicReader),
	)

	// Register metric provider
	otel.SetMeterProvider(metricsProvider)

	err = createCustomMeters(serviceName, serviceVersion, serviceEnvironment)
	if err != nil {
		_ = metricsProvider.Shutdown(context.Background())
		_ = metricExporter.Shutdown(context.Background())
		return nil, fmt.Errorf("failed to create custom meters: %w", err)
	}

	// Trace exporter
	traceExporter, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(exporterEndpoint),
	)
	if err != nil {
		_ = metricsProvider.Shutdown(context.Background())
		_ = metricExporter.Shutdown(context.Background())
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Trace provider
	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(res),
	)

	// Register trace provider
	otel.SetTracerProvider(traceProvider)

	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(propagator)

	return func(ctx context.Context) {
		_ = metricsProvider.Shutdown(ctx)
		_ = metricExporter.Shutdown(ctx)
		_ = traceProvider.Shutdown(ctx)
		_ = traceExporter.Shutdown(ctx)
	}, nil
}

// CacheGetsTotalIncr increases in 1 a metric for tracking cache hits and misses
var CacheGetsTotalIncr func(ctx context.Context, keyPrefix, result string)

// SubtitlesDownloadsTotalIncr increases in 1 a metric for tracking subtitles downloads
var SubtitlesDownloadsTotalIncr func(ctx context.Context)

func createCustomMeters(serviceName, serviceVersion, serviceEnvironment string) error {
	meter := otel.Meter(serviceName)
	var err error
	cacheGetsTotal, err := meter.Int64Counter("cache_gets_total")
	if err != nil {
		return fmt.Errorf("failed to create custom meter: %w", err)
	}
	CacheGetsTotalIncr = func(ctx context.Context, keyPrefix, result string) {
		cacheGetsTotal.Add(ctx, 1, metric2.WithAttributes(
			attribute.String(string(semconv.DeploymentEnvironmentNameKey), serviceEnvironment),
			attribute.String(string(semconv.ServiceVersionKey), serviceVersion),
			attribute.String("key.prefix", keyPrefix),
			attribute.String("result", result),
		))
	}
	subtitlesDownloadsTotal, err := meter.Int64Counter("subtitles_downloads_total")
	if err != nil {
		return fmt.Errorf("failed to create custom meter: %w", err)
	}
	SubtitlesDownloadsTotalIncr = func(ctx context.Context) {
		subtitlesDownloadsTotal.Add(ctx, 1, metric2.WithAttributes(
			attribute.String(string(semconv.DeploymentEnvironmentNameKey), serviceEnvironment),
			attribute.String(string(semconv.ServiceVersionKey), serviceVersion),
		))
	}

	return nil
}

package common

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	slogmulti "github.com/samber/slog-multi"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	traceIDKey = "trace_id"
	spanIDKey  = "span_id"
	// Log is the app global logger
	Log *slog.Logger
)

// InitLogger initializes the app global logger
func InitLogger(serviceName, serviceVersion, serviceEnvironment, exporterEndpoint string) (func(ctx context.Context) error, error) {

	var slogHandler slog.Handler

	ctx := context.Background()

	logExporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(exporterEndpoint),
		otlploggrpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to otlploggrpc.New: %w", err)
	}

	lp := log.NewLoggerProvider(
		log.WithProcessor(
			log.NewBatchProcessor(logExporter),
		),
		log.WithResource(resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(serviceVersion),
			semconv.DeploymentEnvironmentNameKey.String(serviceEnvironment))),
	)

	slogHandler = otelslog.NewHandler("github.com/ogero/stremio-subdivx",
		otelslog.WithLoggerProvider(lp))

	if serviceEnvironment == "lcl" {
		slogHandler = slogmulti.Fanout(
			slogHandler,
			slog.NewTextHandler(os.Stdout, nil),
		)
	}

	Log = slog.New(slogHandler)

	return lp.Shutdown, nil
}

// extractTraceSpanID was taken from https://github.com/samber/slog-chi/blob/679a34d1e3b1c726b040cf7424d797ef8cee48db/middleware.go#L277
func extractTraceSpanID(ctx context.Context) []slog.Attr {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return []slog.Attr{}
	}

	var attrs []slog.Attr
	spanCtx := span.SpanContext()

	if spanCtx.HasTraceID() {
		traceID := trace.SpanFromContext(ctx).SpanContext().TraceID().String()
		attrs = append(attrs, slog.String(traceIDKey, traceID))
	}

	if spanCtx.HasSpanID() {
		spanID := spanCtx.SpanID().String()
		attrs = append(attrs, slog.String(spanIDKey, spanID))
	}

	return attrs
}

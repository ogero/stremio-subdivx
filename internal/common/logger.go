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
)

var (
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

	if serviceEnvironment == "lcl" || serviceEnvironment == "dk" {
		slogHandler = slogmulti.Fanout(
			slogHandler,
			slog.NewTextHandler(os.Stdout, nil),
		)
	}

	Log = slog.New(slogHandler)

	return lp.Shutdown, nil
}

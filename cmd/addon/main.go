package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/ogero/stremio-subdivx/frontend"
	"github.com/ogero/stremio-subdivx/internal"
	"github.com/ogero/stremio-subdivx/internal/cache"
	"github.com/ogero/stremio-subdivx/internal/common"
	"github.com/ogero/stremio-subdivx/pkg/imdb"
	"github.com/ogero/stremio-subdivx/pkg/subdivx"
	slogchi "github.com/samber/slog-chi"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type config struct {
	AddonHost            string `env:"ADDON_HOST" envDefault:"http://127.0.0.1:3593"`
	ServerListenAddr     string `env:"SERVER_LISTEN_ADDR" envDefault:":3593"`
	ServiceName          string `env:"SERVICE_NAME" envDefault:"stremio-subdivx"`
	ServiceEnvironment   string `env:"SERVICE_ENVIRONMENT" envDefault:"lcl"`
	ServiceVersion       string `env:"SERVICE_VERSION" envDefault:"v0.0.3"`
	OtelExporterEndpoint string `env:"OTEL_EXPORTER_ENDPOINT" envDefault:"127.0.0.1:4317"`
}

func main() {

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	cfg, err := env.ParseAs[config]()
	if err != nil {
		panic(fmt.Errorf("failed to env.ParseAs: %w", err))
	}

	loggerShutdown, err := common.InitLogger(cfg.ServiceName, cfg.ServiceVersion, cfg.ServiceEnvironment, cfg.OtelExporterEndpoint)
	if err != nil {
		panic(fmt.Errorf("failed to logger.InitLogger: %w", err))
	}

	err = cache.InitCache(slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		common.Log.Error("Failed to cache.InitCache", "err", err)
		os.Exit(1)
	}

	instrumentationShutdown, err := common.InitInstrumentation(cfg.ServiceName, cfg.ServiceVersion, cfg.ServiceEnvironment, cfg.OtelExporterEndpoint)
	if err != nil {
		common.Log.Error("Failed to common.InitInstrumentation", "err", err)
		os.Exit(1)
	}

	stremioService := internal.NewStremioService(
		imdb.NewStalkrIMDB(),
		subdivx.NewSubdivx())
	app := internal.NewApp(stremioService, cfg.AddonHost)

	distFS, err := fs.Sub(fs.FS(frontend.Dist), "dist")
	if err != nil {
		common.Log.Error("Failed to fs.Sub", "err", err)
	}

	handlersFilter := func(r *http.Request) bool {
		if r.Method == http.MethodGet &&
			(r.URL.Path == "/" ||
				r.URL.Path == "/manifest.json" ||
				strings.HasPrefix(r.URL.Path, "/subtitles/") ||
				strings.HasPrefix(r.URL.Path, "/subdivx/")) {
			return true
		}
		return false
	}

	r := chi.NewRouter()
	r.Use(slogchi.NewWithConfig(common.Log.WithGroup("http"), slogchi.Config{
		Filters: []slogchi.Filter{func(_ middleware.WrapResponseWriter, r *http.Request) bool {
			return handlersFilter(r)
		}},
	}))
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{
			"Content-Type",
			"X-Requested-With",
			"Accept",
			"Accept-Language",
			"Accept-Encoding",
			"Content-Language",
			"Origin",
		},
		MaxAge: 300,
	}))
	r.Handle("GET /manifest.json", otelhttp.WithRouteTag("/manifest.json", http.HandlerFunc(app.ManifestHandler)))
	r.Handle("GET /subtitles/{type}/{id}/*", otelhttp.WithRouteTag("/subtitles/{type}/{id}/*", http.HandlerFunc(app.SubtitlesHandler)))
	r.Handle("GET /subdivx/{id}", otelhttp.WithRouteTag("/subdivx/{id}", http.HandlerFunc(app.SubdivxSubtitleHandler)))
	r.Handle("/*", http.FileServer(http.FS(distFS)))

	// Listen app
	appSrv := &http.Server{
		Addr: cfg.ServerListenAddr,
		Handler: otelhttp.NewHandler(r, "server",
			otelhttp.WithMessageEvents(otelhttp.ReadEvents, otelhttp.WriteEvents),
			otelhttp.WithFilter(func(r *http.Request) bool {
				return handlersFilter(r)
			}),
		),
	}
	go func() {
		common.Log.Info("App started", "Addr", appSrv.Addr, "Host", cfg.AddonHost)
		if err := appSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			common.Log.Error("Failed to http.Server.ListenAndServe", "err", err)
		}
	}()

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := appSrv.Shutdown(ctx); err != nil {
		common.Log.Error("Failed to http.Server.Shutdown", "err", err)
	}

	if err := cache.Close(); err != nil {
		common.Log.Error("Failed to cache.Close", "err", err)
	}

	if instrumentationShutdown != nil {
		instrumentationShutdown(ctx)
	}

	if loggerShutdown != nil {
		_ = loggerShutdown(ctx)
	}

	common.Log.Info("Bye!")
}

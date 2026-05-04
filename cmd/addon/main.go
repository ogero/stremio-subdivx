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
	"github.com/ogero/stremio-subdivx/internal/loki"
	"github.com/ogero/stremio-subdivx/pkg/stremio"
	"github.com/ogero/stremio-subdivx/pkg/subx"
	slogchi "github.com/samber/slog-chi"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type config struct {
	AddonHost            string `env:"ADDON_HOST" envDefault:"http://127.0.0.1:3593"`
	ServerListenAddr     string `env:"SERVER_LISTEN_ADDR" envDefault:":3593"`
	ServiceName          string `env:"SERVICE_NAME" envDefault:"stremio-subdivx"`
	ServiceEnvironment   string `env:"SERVICE_ENVIRONMENT" envDefault:"lcl"`
	ServiceVersion       string `env:"SERVICE_VERSION" envDefault:"v0.0.12"`
	OtelExporterEndpoint string `env:"OTEL_EXPORTER_ENDPOINT" envDefault:"127.0.0.1:4317"`
	LokiHost             string `env:"LOKI_HOST" envDefault:"http://127.0.0.1:3100"`
	StatsWSChannel       string `env:"STATS_WS_CHANNEL" envDefault:"stremio-subdivx:stats"`
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

	stremioManifest := &stremio.Manifest{
		ID:          "ar.xor.subdivx.go",
		Version:     strings.TrimLeft(cfg.ServiceVersion, "v"),
		Name:        "Subdivx",
		Description: "Subdivx subtitles addon",
		Types:       []string{"movie", "series"},
		Catalogs:    []stremio.CatalogItem{},
		IDPrefixes:  []string{"tt"},
		Resources:   []string{"subtitles"},
		BehaviorHints: stremio.BehaviorHints{
			Configurable:          true,
			ConfigurationRequired: true,
		},
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
		cfg.StatsWSChannel,
		subx.NewSubX(),
		loki.NewLoki(cfg.LokiHost),
	)

	go stremioService.StartPollingStats(1 * time.Minute)

	app, err := internal.NewApp(stremioService, stremioManifest, cfg.AddonHost)
	if err != nil {
		common.Log.Error("Failed to internal.NewApp", "err", err)
		os.Exit(1)
	}

	distFS, err := fs.Sub(fs.FS(frontend.Dist), "dist")
	if err != nil {
		common.Log.Error("Failed to fs.Sub", "err", err)
	}

	r := chi.NewRouter()
	r.Use(slogchi.NewWithConfig(common.Log.WithGroup("http"), slogchi.Config{
		Filters: []slogchi.Filter{func(_ middleware.WrapResponseWriter, r *http.Request) bool {
			return handlersFilter(r)
		}},
	}))
	r.Use(middleware.Recoverer)
	r.Use(otelRoutePattern)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "OPTIONS"},
		AllowedHeaders: []string{
			"Content-Type",
			"X-Requested-With",
			"Accept",
			"Accept-Language",
			"Accept-Encoding",
			"Content-Language",
			"Origin",
			"Sec-WebSocket-Version",
			"Sec-WebSocket-Key",
			"Sec-WebSocket-Extensions",
			"Upgrade",
			"Connection",
		},
		MaxAge: 300,
	}))
	r.Handle("GET /manifest.json", http.HandlerFunc(app.ManifestHandler))
	r.Handle("GET /{userConfig}/manifest.json", http.HandlerFunc(app.ManifestHandler))
	r.Handle("GET /{userConfig}/subtitles/{type}/{id}/*", http.HandlerFunc(app.SubtitlesHandler))
	r.Handle("GET /{userConfig}/subx/{id}", http.HandlerFunc(app.SubXSubtitleHandler))
	r.Handle("GET /ws", http.HandlerFunc(app.WebsocketHandler))
	r.Handle("GET /configure", spaIndexHandler(distFS))
	r.Handle("GET /{userConfig}/configure", spaIndexHandler(distFS))
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

func handlersFilter(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}

	path := r.URL.Path
	if path == "/" || path == "/ws" || path == "/manifest.json" || path == "/configure" {
		return true
	}

	_, pathAfterUserConfig, ok := strings.Cut(strings.TrimPrefix(path, "/"), "/")
	if !ok {
		return false
	}

	return pathAfterUserConfig == "manifest.json" ||
		pathAfterUserConfig == "configure" ||
		strings.HasPrefix(pathAfterUserConfig, "subtitles/") ||
		strings.HasPrefix(pathAfterUserConfig, "subx/")
}

func otelRoutePattern(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)

		route := r.Pattern
		if route == "" {
			route = chi.RouteContext(r.Context()).RoutePattern()
		}
		if route == "" {
			return
		}

		r.Pattern = route
		trace.SpanFromContext(r.Context()).SetAttributes(attribute.String("http.route", route))
	})
}

func spaIndexHandler(distFS fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		index, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(index)
	}
}

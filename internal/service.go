package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/centrifugal/centrifuge"
	"github.com/ogero/stremio-subdivx/internal/cache"
	"github.com/ogero/stremio-subdivx/internal/common"
	"github.com/ogero/stremio-subdivx/internal/loki"
	"github.com/ogero/stremio-subdivx/pkg/imdb"
	"github.com/ogero/stremio-subdivx/pkg/subdivx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Subtitles struct holds information about subtitles, including their IDs, language, and the year of the content they are associated with.
type Subtitles struct {
	// IDs is a list of subtitle IDs.
	IDs []string
	// Lang is the language of the subtitles.
	Lang string
	// Year is the year of the content the subtitles are for.
	Year int
}

// Stats represents statistical data including search and download counts in the last 24 hours and instant title information.
type Stats struct {
	// SearchesCount24 represents the number of searches performed in the last 24 hours.
	SearchesCount24 int `json:"searchesCount24"`
	// DownloadsCount24 represents the number of downloads within the last 24 hours.
	DownloadsCount24 int `json:"downloadsCount24"`
	// TitleInstant holds the title information for immediate reporting or broadcasting in statistical data.
	TitleInstant string `json:"titleInstant"`
}

// StremioService defines methods for retrieving subtitles, either by IMDb ID, season, and episode, or by a specific Subdivx ID.
type StremioService interface {
	// Handler handles incoming HTTP requests via a websocket handler
	http.Handler
	// GetSubtitles retrieves subtitles for a given IMDb ID, season, and episode.
	GetSubtitles(ctx context.Context, imdbID string, season int, episode int) (*Subtitles, error)
	// GetSubtitle retrieves a specific subtitle by its Subdivx ID.
	GetSubtitle(ctx context.Context, subdivxID string) ([]byte, error)
	// BroadcastStats updates and publishes statistical data to a websocket channel.
	// Accepts a function to modify stats and returns an error if updating or publishing fails.
	BroadcastStats(statsUpdater func(stats *Stats) error) error
	// StartPollingStats begins the periodic fetching and broadcasting of statistical data at the specified interval.
	StartPollingStats(interval time.Duration)
}

type stremioService struct {
	statsWebsocketChannel string
	imdb                  imdb.IMDB
	subdivx               subdivx.Subdivx
	loki                  loki.Loki

	node             *centrifuge.Node
	websocketHandler *centrifuge.WebsocketHandler
	statsMutex       *sync.Mutex
	stats            Stats
}

// NewStremioService creates a new instance of StremioService with the provided IMDb and Subdivx clients.
func NewStremioService(statsWebsocketChannel string, imdb imdb.IMDB, subdivx subdivx.Subdivx, loki loki.Loki) StremioService {
	svc := &stremioService{
		statsWebsocketChannel: statsWebsocketChannel,
		imdb:                  imdb,
		subdivx:               subdivx,
		loki:                  loki,

		statsMutex: &sync.Mutex{},
	}

	node, err := centrifuge.New(centrifuge.Config{})
	if err != nil {
		common.Log.Error("Failed to centrifuge.New", "err", err)
		os.Exit(1)
	}
	svc.node = node

	node.OnConnecting(func(ctx context.Context, e centrifuge.ConnectEvent) (centrifuge.ConnectReply, error) {
		return centrifuge.ConnectReply{}, nil
	})

	node.OnConnect(func(client *centrifuge.Client) {
		client.OnSubscribe(func(e centrifuge.SubscribeEvent, cb centrifuge.SubscribeCallback) {
			if e.Channel != statsWebsocketChannel {
				cb(centrifuge.SubscribeReply{}, centrifuge.ErrorPermissionDenied)
				return
			}

			cb(centrifuge.SubscribeReply{
				Options: centrifuge.SubscribeOptions{},
			}, nil)

			// Todo: Avoid broadcasting to all clients
			go func() {
				err := svc.BroadcastStats(func(data *Stats) error { return nil })
				if err != nil {
					common.Log.Warn("Failed to internal.StremioService.BroadcastStats", "err", err)
				}
			}()
		})

	})

	if err := node.Run(); err != nil {
		common.Log.Error("Failed to centrifuge.Node.Run", "err", err)
		os.Exit(1)
	}

	websocketHandler := centrifuge.NewWebsocketHandler(node, centrifuge.WebsocketConfig{
		ReadBufferSize:     1024,
		UseWriteBufferPool: true,
	})
	svc.websocketHandler = websocketHandler

	return svc
}

// GetSubtitles retrieves subtitles for a given IMDb ID, season, and episode.
// It uses caching to improve performance and reduce API calls.
func (s *stremioService) GetSubtitles(ctx context.Context, imdbID string, season int, episode int) (*Subtitles, error) {

	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("").Start(ctx, "internal.StremioService.GetSubtitles")
	defer span.End()

	cacheResult := "hit"
	cacheKey := fmt.Sprintf("imdb.title : %s", imdbID)
	cacheTTL := 48 * time.Hour
	imdbTitle, err := cache.Memoize[imdb.Title](cacheKey, cacheTTL, func() (*imdb.Title, error) {

		cacheResult = "miss"
		title, err := s.imdb.GetTitle(ctx, imdbID)
		if err != nil {
			return nil, fmt.Errorf("failed to imdb.IMDB.GetTitle: %w", err)
		}

		return title, nil
	})
	span.SetAttributes(attribute.String("cache.imdb.title.result", cacheResult))
	common.CacheGetsTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("key.prefix", "imdb.title"),
		attribute.String("result", cacheResult),
	))
	if err != nil {
		return nil, err
	}

	span.SetAttributes(attribute.String("imdb.id", imdbID))
	span.SetAttributes(attribute.String("imdb.title", imdbTitle.Name))
	span.SetAttributes(attribute.Int("imdb.season", season))
	span.SetAttributes(attribute.Int("imdb.episode", episode))

	go func() {
		err := s.BroadcastStats(func(data *Stats) error {
			data.TitleInstant = imdbTitle.Name
			return nil
		})
		if err != nil {
			common.Log.WarnContext(ctx, "Failed to internal.StremioService.BroadcastStats", "err", err)
		}
	}()

	subdivxSearchTerm := imdbTitle.Name
	if season != 0 && episode != 0 {
		subdivxSearchTerm = fmt.Sprintf("%s S%02dE%02d", imdbTitle.Name, season, episode)
	}

	cacheResult = "hit"
	cacheKey = fmt.Sprintf("subdivx.subtitles : %s", subdivxSearchTerm)
	cacheTTL = 24 * time.Hour
	subdivxSubtitles, err := cache.Memoize[subdivx.Subtitles](cacheKey, cacheTTL, func() (*subdivx.Subtitles, error) {

		cacheResult = "miss"
		token, err := s.subdivx.GetToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to subdivx.Subdivx.GetToken: %w", err)
		}

		subtitles, err := s.subdivx.GetSubtitles(ctx, token, subdivxSearchTerm)
		if err != nil {
			return nil, fmt.Errorf("failed to subdivx.Subdivx.GetSubtitles: %w", err)
		}

		return subtitles, nil
	})
	span.SetAttributes(attribute.String("cache.subdivx.subtitles.result", cacheResult))
	common.CacheGetsTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("key.prefix", "subdivx.subtitles"),
		attribute.String("result", cacheResult),
	))
	if err != nil {
		return nil, err
	}
	common.Log.InfoContext(ctx, "Found subtitles", "ids", subdivxSubtitles.IDs)
	span.SetAttributes(attribute.Int("subdivx.total-records", subdivxSubtitles.TotalRecords))
	span.SetAttributes(attribute.Int("subdivx.ids-count", len(subdivxSubtitles.IDs)))

	ids := make([]string, 0, len(subdivxSubtitles.IDs))
	for _, subdivxSubtitleID := range subdivxSubtitles.IDs {
		ids = append(ids, strconv.Itoa(subdivxSubtitleID))
	}

	return &Subtitles{
		IDs:  ids,
		Lang: "spa",
		Year: imdbTitle.Year,
	}, nil

}

// GetSubtitle retrieves a specific subtitle by its Subdivx ID.
func (s *stremioService) GetSubtitle(ctx context.Context, subdivxID string) ([]byte, error) {

	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("").Start(ctx, "internal.StremioService.GetSubtitle")
	defer span.End()

	subtitle, err := s.subdivx.GetSubtitle(ctx, subdivxID)
	if err != nil {
		return nil, fmt.Errorf("failed to subdivx.Subdivx.GetSubtitle: %w", err)
	}
	common.Log.WithGroup("file").InfoContext(ctx, "Got SRT", "name", subtitle.Name, "size", len(subtitle.Data))

	return subtitle.Data, nil
}

// BroadcastStats updates and publishes statistical data to a websocket channel.
// Accepts a function to modify stats and returns an error if updating or publishing fails.
func (s *stremioService) BroadcastStats(statsUpdater func(stats *Stats) error) error {
	stats, err := func() (Stats, error) {
		s.statsMutex.Lock()
		defer s.statsMutex.Unlock()
		err := statsUpdater(&s.stats)
		if err != nil {
			return Stats{}, err
		}
		return s.stats, nil
	}()
	if err != nil {
		return fmt.Errorf("failed to statsUpdater: %w", err)
	}

	b, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("failed to json.Marshal: %w", err)
	}

	_, err = s.node.Publish(s.statsWebsocketChannel, b, nil...)
	if err != nil {
		return fmt.Errorf("failed to centrifuge.Node.Publish: %w", err)
	}

	return nil
}

// StartPollingStats begins the periodic fetching and broadcasting of statistical data at the specified interval.
func (s *stremioService) StartPollingStats(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for ; true; <-ticker.C {
		searches, err := s.loki.GetSearches24()
		if err != nil {
			common.Log.Error("failed to get loki.Loki.GetSearches24", "err", err)
		}
		downloads, err := s.loki.GetDownloads24()
		if err != nil {
			common.Log.Error("failed to get loki.Loki.GetDownloads24", "err", err)
		}
		err = s.BroadcastStats(func(stats *Stats) error {
			if searches != 0 {
				stats.SearchesCount24 = searches
			}
			if downloads != 0 {
				stats.DownloadsCount24 = downloads
			}
			return nil
		})
		if err != nil {
			common.Log.Warn("failed to internal.StremioService.BroadcastStats", "err", err)
		}
	}
}

// ServeHTTP handles incoming HTTP requests via a websocket handler
func (s *stremioService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	newCtx := centrifuge.SetCredentials(ctx, &centrifuge.Credentials{})
	r = r.WithContext(newCtx)

	s.websocketHandler.ServeHTTP(w, r)
}

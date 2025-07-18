package internal

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ogero/stremio-subdivx/internal/cache"
	"github.com/ogero/stremio-subdivx/internal/common"
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

// StremioService defines methods for retrieving subtitles, either by IMDb ID, season, and episode, or by a specific Subdivx ID.
type StremioService interface {
	// GetSubtitles retrieves subtitles for a given IMDb ID, season, and episode.
	GetSubtitles(ctx context.Context, imdbID string, season int, episode int) (*Subtitles, error)
	// GetSubtitle retrieves a specific subtitle by its Subdivx ID.
	GetSubtitle(ctx context.Context, subdivxID string) ([]byte, error)
}

type stremioService struct {
	imdb    imdb.IMDB
	subdivx subdivx.Subdivx
}

// NewStremioService creates a new instance of StremioService with the provided IMDb and Subdivx clients.
func NewStremioService(imdb imdb.IMDB, subdivx subdivx.Subdivx) StremioService {
	return &stremioService{
		imdb:    imdb,
		subdivx: subdivx,
	}
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

package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/centrifugal/centrifuge"
	"github.com/ogero/stremio-subdivx/internal/cache"
	"github.com/ogero/stremio-subdivx/internal/common"
	"github.com/ogero/stremio-subdivx/internal/loki"
	"github.com/ogero/stremio-subdivx/pkg/subx"
	"github.com/wlynxg/chardet"
	"github.com/wlynxg/chardet/consts"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// Subtitles struct holds information about subtitles, including their IDs, language, and the year of the content they are associated with.
type Subtitles struct {
	// IDs is a list of subtitle IDs.
	IDs []string
	// Lang is the language of the subtitles.
	Lang string
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

type StremioService struct {
	statsWebsocketChannel string
	subx                  *subx.SubX
	loki                  loki.Loki

	node             *centrifuge.Node
	websocketHandler *centrifuge.WebsocketHandler
	statsMutex       *sync.Mutex
	stats            Stats
}

// NewStremioService creates a new instance of StremioService with the provided SubX client.
func NewStremioService(statsWebsocketChannel string, subxClient *subx.SubX, loki loki.Loki) *StremioService {
	svc := &StremioService{
		statsWebsocketChannel: statsWebsocketChannel,
		subx:                  subxClient,
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

// GetSubtitles retrieves subtitles for a given title type, IMDb ID, season, and episode; filename is used to sort results by relevance
// It uses caching to improve performance and reduce API calls.
func (s *StremioService) GetSubtitles(ctx context.Context, subxAPIKey string, titleType string, imdbID string, season int, episode int, filename string) (*Subtitles, error) {

	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("").Start(ctx, "internal.StremioService.GetSubtitles")
	defer span.End()

	span.SetAttributes(attribute.String("imdb.id", imdbID))
	span.SetAttributes(attribute.Int("imdb.season", season))
	span.SetAttributes(attribute.Int("imdb.episode", episode))

	searchLabel := imdbID
	if titleType == "movie" {
		searchLabel = fmt.Sprintf("%s movie", imdbID)
	} else if season > 0 && episode > 0 {
		searchLabel = fmt.Sprintf("%s S%02dE%02d", imdbID, season, episode)
	}

	cacheResult := "hit"
	cacheKey := fmt.Sprintf("subx.subtitles : %s : %s : %d : %d", titleType, imdbID, season, episode)
	cacheTTL := 24 * time.Hour
	subxSubtitles, err := cache.Memoize[subx.Subtitles](cacheKey, cacheTTL, func() (*subx.Subtitles, error) {

		cacheResult = "miss"

		common.Log.InfoContext(ctx, "Searching SubX subtitles", "imdb_id", imdbID, "type", titleType, "season", season, "episode", episode)

		subtitles, err := s.subx.SearchSubtitles(ctx, subxAPIKey, subx.SearchParams{
			IMDBID: imdbID,
			Limit:  50,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to subx.SubX.SearchSubtitles: %w", err)
		}

		if titleType == "series" && season > 0 && episode > 0 {
			filteredSubtitles := make([]*subx.Subtitle, 0, len(subtitles.Subtitles))
			for _, subtitle := range subtitles.Subtitles {
				if subtitle.Season == season && subtitle.Episode == episode {
					filteredSubtitles = append(filteredSubtitles, subtitle)
				}
			}
			subtitles.TotalRecords = len(filteredSubtitles)
			subtitles.Subtitles = filteredSubtitles
		}

		return subtitles, nil
	})
	span.SetAttributes(attribute.String("cache.subx.subtitles.result", cacheResult))
	common.CacheGetsTotalIncr(ctx, "subx.subtitles", cacheResult)
	if err != nil {
		return nil, err
	}
	span.SetAttributes(attribute.Int("subx.total-records", subxSubtitles.TotalRecords))
	span.SetAttributes(attribute.Int("subx.ids-count", len(subxSubtitles.Subtitles)))

	type ScoredSubtitle struct {
		ID    string
		Score int
	}

	subxScoredSubtitles := make([]ScoredSubtitle, 0, len(subxSubtitles.Subtitles))
	for _, subxSubtitle := range subxSubtitles.Subtitles {
		subxScoredSubtitle := ScoredSubtitle{
			ID:    subxSubtitle.ID,
			Score: subxSubtitle.Score(filename),
		}
		subxScoredSubtitles = append(subxScoredSubtitles, subxScoredSubtitle)
	}
	sort.Slice(subxScoredSubtitles, func(i, j int) bool {
		return subxScoredSubtitles[i].Score > subxScoredSubtitles[j].Score
	})

	ids := make([]string, len(subxScoredSubtitles))
	scores := make([]int, len(subxScoredSubtitles))
	for i, item := range subxScoredSubtitles {
		ids[i] = item.ID
		scores[i] = item.Score
	}
	common.Log.InfoContext(ctx, "Found subtitles", "title", searchLabel, "ids", ids, "scores", scores)

	go func() {
		titleInstant := searchLabel
		if len(subxSubtitles.Subtitles) > 0 && subxSubtitles.Subtitles[0].Title != "" {
			titleInstant = subxSubtitles.Subtitles[0].Title
			if titleType == "series" && season > 0 && episode > 0 {
				titleInstant = fmt.Sprintf("%s S%02dE%02d", titleInstant, season, episode)
			}
		}

		err := s.BroadcastStats(func(data *Stats) error {
			data.TitleInstant = titleInstant
			return nil
		})
		if err != nil {
			common.Log.WarnContext(ctx, "Failed to internal.StremioService.BroadcastStats", "err", err)
		}
	}()

	return &Subtitles{
		IDs:  ids,
		Lang: "spa",
	}, nil

}

// GetSubtitle retrieves a specific subtitle by its SubX ID.
func (s *StremioService) GetSubtitle(ctx context.Context, subxAPIKey string, subxID string) ([]byte, error) {

	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("").Start(ctx, "internal.StremioService.GetSubtitle")
	defer span.End()

	common.SubtitlesDownloadsTotalIncr(ctx)

	subtitle, err := s.subx.DownloadSubtitle(ctx, subxAPIKey, subxID)
	if err != nil {
		return nil, fmt.Errorf("failed to subx.SubX.DownloadSubtitle: %w", err)
	}

	fileEncoding := chardet.Detect(subtitle.Data).Encoding
	common.Log.WithGroup("file").InfoContext(ctx, "Got SRT", "name", subtitle.Name, "encoding", fileEncoding, "size", len(subtitle.Data))

	var decoder *encoding.Decoder
	switch fileEncoding {
	case consts.Windows1252:
		decoder = charmap.Windows1252.NewDecoder()
	case consts.ISO88591:
		decoder = charmap.ISO8859_1.NewDecoder()
	}

	if decoder != nil {
		tr := transform.NewReader(bytes.NewReader(subtitle.Data), decoder)
		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("failed to io.ReadAll when transforming subtitle encoding: %w", err)
		}
		return data, nil
	}

	return subtitle.Data, nil
}

// BroadcastStats updates and publishes statistical data to a websocket channel.
// Accepts a function to modify stats and returns an error if updating or publishing fails.
func (s *StremioService) BroadcastStats(statsUpdater func(stats *Stats) error) error {
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
func (s *StremioService) StartPollingStats(interval time.Duration) {
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
func (s *StremioService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	newCtx := centrifuge.SetCredentials(ctx, &centrifuge.Credentials{})
	r = r.WithContext(newCtx)

	s.websocketHandler.ServeHTTP(w, r)
}

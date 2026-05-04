package internal

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/ogero/stremio-subdivx/internal/common"
	"github.com/ogero/stremio-subdivx/pkg/stremio"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// App represents the main application structure that holds the Stremio service and addon host information.
type App struct {
	StremioService  *StremioService
	StremioManifest *stremio.Manifest
	AddonHost       string
}

/*
NewApp creates a new instance of the App struct.

Parameters:
  - stremioService: The service used to interact with Stremio.
  - stremioManifest: The manifest used to interact with Stremio.
  - addonHost: The host address for the addon.

Returns:
  - A pointer to the newly created App instance.
*/
func NewApp(stremioService *StremioService, stremioManifest *stremio.Manifest, addonHost string) (*App, error) {
	return &App{
		StremioService:  stremioService,
		StremioManifest: stremioManifest,
		AddonHost:       addonHost,
	}, nil
}

/*
ManifestHandler serves the manifest for the addon.

This method writes the manifest as a JSON response to the HTTP writer.
*/
func (a *App) ManifestHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)

	common.Log.DebugContext(ctx, "ManifestHandler")

	w.Header().Set("Content-Type", "application/json")

	manifest := *a.StremioManifest
	if apiKeyFromUserConfig(chi.URLParam(r, "userConfig")) != "" {
		manifest.BehaviorHints.ConfigurationRequired = false
	}

	b, _ := json.Marshal(manifest)
	_, err := w.Write(b)
	if err != nil {
		common.Log.ErrorContext(ctx, "Failed to write response", "err", err)
		span.RecordError(err)
		return
	}

}

func apiKeyFromUserConfig(userConfig string) string {
	if userConfig == "" {
		return ""
	}

	decoded, err := base64.RawURLEncoding.DecodeString(userConfig)
	if err != nil {
		decoded, err = base64.URLEncoding.DecodeString(userConfig)
		if err != nil {
			return ""
		}
	}

	var config struct {
		APIKey string `json:"apiKey"`
	}
	if err = json.Unmarshal(decoded, &config); err != nil {
		return ""
	}

	return strings.TrimSpace(config.APIKey)
}

/*
SubtitlesHandler handles requests for subtitles.

This method validates the request parameters, fetches subtitles from the Stremio service, and writes them as a JSON response.
*/
func (a *App) SubtitlesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)

	common.Log.DebugContext(ctx, "SubtitlesHandler")

	paramsType := chi.URLParam(r, "type")
	if err := common.ValidateSubtitleType(paramsType); err != nil {
		common.Log.WarnContext(ctx, "Failed to common.ValidateSubtitleType", "err", err)
		span.RecordError(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	span.SetAttributes(attribute.String("params.type", paramsType))

	paramsID, err := url.PathUnescape(chi.URLParam(r, "id"))
	if err != nil {
		common.Log.WarnContext(ctx, "Failed to url.PathUnescape", "err", err)
		span.RecordError(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	span.SetAttributes(attribute.String("param.id", paramsID))

	var imdbID string
	var seasonNumber, episodeNumber int

	paramsIds := strings.Split(paramsID, ":")
	imdbID = paramsIds[0]
	if err = common.ValidateIMDBTitleID(imdbID); err != nil {
		common.Log.WarnContext(ctx, "Failed to common.ValidateIMDBTitleID", "err", err)
		span.RecordError(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(paramsIds) == 3 {
		seasonNumber, err = strconv.Atoi(paramsIds[1])
		if err != nil {
			common.Log.WarnContext(ctx, "Failed to convert season to a number", "err", err)
		}
		episodeNumber, err = strconv.Atoi(paramsIds[2])
		if err != nil {
			common.Log.WarnContext(ctx, "Failed to convert episode to a number", "err", err)
		}
	}

	paramsWildcard := chi.URLParam(r, "*")
	var queryFilename string
	if queryValues, err := url.ParseQuery(paramsWildcard); err != nil {
		common.Log.WarnContext(ctx, "Failed to url.ParseQuery", "err", err)
		span.RecordError(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	} else if queryFilename = queryValues.Get("filename"); queryFilename == "" {
		common.Log.WarnContext(ctx, "Failed to url.Values.Get(filename)", "err", fmt.Errorf("filename not found"))
	}

	userConfig := chi.URLParam(r, "userConfig")
	apiKey := apiKeyFromUserConfig(userConfig)
	if apiKey == "" {
		common.Log.WarnContext(ctx, "Failed to apiKeyFromUserConfig", "err", fmt.Errorf("api key not found"))
		span.RecordError(fmt.Errorf("api key not found"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	subtitles, err := a.StremioService.GetSubtitles(ctx, apiKey, paramsType, imdbID, seasonNumber, episodeNumber, queryFilename)
	if err != nil {
		common.Log.ErrorContext(ctx, "Failed to StremioService.GetSubtitles", "err", err)
		span.RecordError(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := stremio.Subtitles{
		Subtitles: make([]stremio.Subtitle, 0, len(subtitles.IDs)),
	}
	for _, id := range subtitles.IDs {
		response.Subtitles = append(response.Subtitles, stremio.Subtitle{
			ID:   id,
			Lang: subtitles.Lang,
			URL:  fmt.Sprintf("%s/%s/subx/%s", a.AddonHost, userConfig, id),
		})
	}

	w.Header().Set("CDN-Cache-Control", "public, max-age=600")
	w.Header().Set("Cache-Control", "public, max-age=600")
	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		common.Log.ErrorContext(ctx, "Failed to write response", "err", err)
		span.RecordError(err)
		return
	}
}

/*
SubXSubtitleHandler handles requests for a specific subtitle by ID.

This method validates the subtitle ID, fetches the subtitle data, and writes it to the response with the appropriate content type.
*/
func (a *App) SubXSubtitleHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)

	common.Log.DebugContext(ctx, "SubXSubtitleHandler")

	apiKey := apiKeyFromUserConfig(chi.URLParam(r, "userConfig"))
	if apiKey == "" {
		common.Log.WarnContext(ctx, "Failed to apiKeyFromUserConfig", "err", fmt.Errorf("api key not found"))
		span.RecordError(fmt.Errorf("api key not found"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	paramsID := chi.URLParam(r, "id")
	if err := common.ValidateSubXSubtitleID(paramsID); err != nil {
		common.Log.WarnContext(ctx, "Failed to common.ValidateSubXSubtitleID", "err", err)
		span.RecordError(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	span.SetAttributes(attribute.String("param.id", paramsID))

	data, err := a.StremioService.GetSubtitle(ctx, apiKey, paramsID)
	if err != nil {
		common.Log.ErrorContext(ctx, "Failed to StremioService.GetSubtitle", "err", err)
		span.RecordError(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/force-download")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.srt\"", paramsID))
	w.Header().Set("CDN-Cache-Control", "public, max-age=1296000")
	w.Header().Set("Cache-Control", "public, max-age=1296000")

	_, err = w.Write(data)
	if err != nil {
		common.Log.ErrorContext(ctx, "Failed to write response", "err", err)
		span.RecordError(err)
		return
	}
}

// WebsocketHandler handles WebSocket connections
func (a *App) WebsocketHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	common.Log.DebugContext(ctx, "WebsocketHandler")

	a.StremioService.ServeHTTP(w, r)
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	imdblib "github.com/StalkR/imdb"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/ogero/stremio-subdivx/frontend"
	"github.com/ogero/stremio-subdivx/internal/cache"
	"github.com/ogero/stremio-subdivx/internal/config"
	"github.com/ogero/stremio-subdivx/pkg/imdb"
	"github.com/ogero/stremio-subdivx/pkg/stremio"
	"github.com/ogero/stremio-subdivx/pkg/subdivx"
	"github.com/wlynxg/chardet"
	"github.com/wlynxg/chardet/consts"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

var manifest = stremio.Manifest{
	ID:          "ar.xor.subdivx.go",
	Version:     "0.0.1",
	Name:        "Subdivx",
	Description: "Subdivx subtitles addon",
	Types:       []string{"movie", "series"},
	Catalogs:    []stremio.CatalogItem{},
	IDPrefixes:  []string{"tt"},
	Resources:   []string{"subtitles"},
}

func main() {

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	distFS, err := fs.Sub(fs.FS(frontend.Dist), "dist")
	if err != nil {
		log.Fatal(fmt.Errorf("failed to fs.Sub: %w", err))
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
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
	r.Get("/manifest.json", manifestHandler)
	r.Get("/subtitles/{type}/{id}/*", subtitlesHandler)
	r.Get("/subdivx/{id}", subdivxSRTHandler)
	r.Handle("/*", http.FileServer(http.FS(distFS)))

	// Listen
	srv := &http.Server{
		Addr:    config.ServerListenAddr,
		Handler: r,
	}
	go func() {
		log.Println("Listening on", config.ServerListenAddr)
		log.Println("Install at", fmt.Sprintf("%s/manifest.json", config.AddonHost))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Println("Failed to http.Server.ListenAndServe:", err)
		}
	}()

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Println("Failed to http server shutdown:", err)
	}

	if err := cache.Close(); err != nil {
		log.Println("Failed to cache.Close:", err)
	}

	log.Println("Bye!")
}

func manifestHandler(w http.ResponseWriter, _ *http.Request) {
	jr, _ := json.Marshal(manifest)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(jr)
}

func subtitlesHandler(w http.ResponseWriter, r *http.Request) {
	paramsType := chi.URLParam(r, "type")
	if paramsType != "movie" && paramsType != "series" {
		log.Println("Invalid subtitles type:", paramsType)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	paramsID := chi.URLParam(r, "id")
	if !strings.HasPrefix(paramsID, "tt") {
		log.Println("Invalid subtitle id:", paramsID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	paramsID, err := url.PathUnescape(paramsID)
	if err != nil {
		log.Println("Failed to url.PathUnescape:", err)
	}

	var imdbID string
	var seasonNumber, episodeNumber int
	paramsIds := strings.Split(paramsID, ":")
	imdbID = paramsIds[0]
	if len(paramsIds) > 1 {
		seasonNumber, _ = strconv.Atoi(paramsIds[1])
		episodeNumber, _ = strconv.Atoi(paramsIds[2])
	}
	log.Println("Stremio requested imdb id:", imdbID, " season number:", seasonNumber, " and episode number:", episodeNumber)

	cacheOp := "hit"
	imdbTitle, err := cache.Memoize[imdblib.Title](fmt.Sprintf("imdb.title : %s", imdbID), 48*time.Hour, func(s string) (*imdblib.Title, error) {

		cacheOp = "miss"
		imdbTitle, err := imdb.FetchTitle(imdbID)
		if err != nil {
			return nil, fmt.Errorf("failed to FetchIMDBTitle: %w", err)
		}

		return imdbTitle, nil
	})
	if err != nil {
		log.Println("Failed to fetch imdb title:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Println("IMDB cache op:", cacheOp)

	subdivxSearchTerm := imdbTitle.Name
	if seasonNumber != 0 && episodeNumber != 0 {
		subdivxSearchTerm = fmt.Sprintf("%s S%02dE%02d", imdbTitle.Name, seasonNumber, episodeNumber)
	}

	cacheOp = "hit"
	searchTitleResponse, err := cache.Memoize[subdivx.SearchTitleResponse](fmt.Sprintf("subdivx.title : %s", subdivxSearchTerm), 24*time.Hour, func(s string) (*subdivx.SearchTitleResponse, error) {

		cacheOp = "miss"
		subdivxToken, subdivxCookie, err := subdivx.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to subdivx.Token: %w", err)
		}
		log.Println("Got subdivx token:", subdivxToken, " and cookie:", subdivxCookie)

		log.Println("Fetching subdivx subtitles for:", subdivxSearchTerm)
		searchTitleResponse, err := subdivx.SearchTitle(subdivxToken, subdivxCookie, subdivxSearchTerm)
		if err != nil {
			return nil, fmt.Errorf("failed to subdivx.SearchTitle: %w", err)
		}
		log.Println("Got subdivx subs:", searchTitleResponse.ITotalRecords, " (", func() string {
			var subs []string
			for _, sub := range searchTitleResponse.AaData {
				subs = append(subs, strconv.Itoa(sub.ID))
			}
			return strings.Join(subs, ", ")
		}(), ")")

		return searchTitleResponse, nil
	})
	if err != nil {
		log.Println("Failed to fetch subdivx subs:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Println("Subdivx cache op:", cacheOp)

	subs := make([]stremio.Subtitle, 0, len(searchTitleResponse.AaData))
	for _, sub := range searchTitleResponse.AaData {
		subs = append(subs, stremio.Subtitle{
			ID:   strconv.Itoa(sub.ID),
			Lang: `spa`,
			URL:  fmt.Sprintf("%s/subdivx/%d", config.AddonHost, sub.ID),
		})
	}

	if imdbTitle.Year < time.Now().Year()-1 && len(subs) > 1 {
		w.Header().Set("CDN-Cache-Control", "public, max-age=1296000")
		w.Header().Set("Cache-Control", "public, max-age=1296000")
	} else {
		w.Header().Set("CDN-Cache-Control", "public, max-age=120")
		w.Header().Set("Cache-Control", "public, max-age=120")
	}

	response := map[string]interface{}{"subtitles": subs}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func subdivxSRTHandler(w http.ResponseWriter, r *http.Request) {

	paramsID := chi.URLParam(r, "id")
	if _, err := strconv.Atoi(paramsID); err != nil {
		log.Println("Invalid subtitle id:", paramsID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	srtByID, err := subdivx.DownloadSRTByID(paramsID)
	if err != nil {
		log.Println("Failed to subdivx.DownloadSRTByID:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", fmt.Sprintf("text/plain; charset=%s", consts.UTF8))
	w.Header().Set("CDN-Cache-Control", "public, max-age=1296000")
	w.Header().Set("Cache-Control", "public, max-age=1296000")

	switch chardet.Detect(srtByID).Encoding {
	case consts.UTF8:
		_, err = w.Write(srtByID)
	case consts.ISO88591:
		tr := transform.NewReader(bytes.NewReader(srtByID), charmap.ISO8859_1.NewDecoder())
		_, err = io.Copy(w, tr)
	default:
		_, err = w.Write(srtByID)
	}

	if err != nil {
		log.Println("Failed to send response:", err)
		return
	}

}

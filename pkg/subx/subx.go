package subx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gen2brain/go-unarr"
	"github.com/ogero/stremio-subdivx/pkg/transport"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultBaseURL         = "https://subx-api.duckdns.org"
	defaultSearchLimit     = 50
	maxSubtitleArchiveSize = 5 * 1024 * 1024
	maxSubtitleFileSize    = 500 * 1024
	maxErrorBodySize       = 4 * 1024
)

// Subtitles holds the total number of records and the matching subtitles.
type Subtitles struct {
	TotalRecords int
	Subtitles    []*Subtitle
}

// Subtitle holds a single SubX subtitle search result.
type Subtitle struct {
	ID               string
	VideoType        string
	Title            string
	Season           int
	Episode          int
	IMDBID           string
	Description      string
	UploaderName     string
	PostedAt         string
	Downloads        int
	DescriptionWords []string
}

// SubtitleContents holds content of a subtitle.
type SubtitleContents struct {
	Name string
	Data []byte
}

// SearchParams represents SubX subtitle search filters.
type SearchParams struct {
	Title  string
	IMDBID string
	Limit  int
}

// NewSubX creates a new instance of the SubX service.
func NewSubX() *SubX {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100

	rt := transport.NewModifyHeadersRoundTripper(t,
		transport.WithAcceptLanguage("es-AR,es;q=0.9,en;q=0.8"),
		transport.WithUserAgent("stremio-subdivx.xor.ar"),
	)

	return &SubX{
		HttpClient: &http.Client{
			Timeout:   time.Second * 10,
			Transport: rt,
		},
		BaseURL:     defaultBaseURL,
		SearchLimit: defaultSearchLimit,
	}
}

type SubX struct {
	HttpClient  *http.Client
	BaseURL     string
	SearchLimit int
}

// SearchSubtitles fetches subtitles using explicit SubX search filters.
func (s *SubX) SearchSubtitles(ctx context.Context, apiKey string, params SearchParams) (*Subtitles, error) {
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("").Start(ctx, "subx.SubX.SearchSubtitles")
	defer span.End()

	if apiKey == "" {
		return nil, fmt.Errorf("api key is empty")
	}

	if s.HttpClient == nil {
		return nil, fmt.Errorf("http client is nil")
	}

	endpoint, err := url.Parse(strings.TrimRight(s.BaseURL, "/") + "/api/subtitles/search")
	if err != nil {
		return nil, fmt.Errorf("failed to url.Parse: %w", err)
	}

	query := endpoint.Query()
	if title := strings.TrimSpace(params.Title); title != "" {
		query.Set("title", title)
	}
	if imdbID := strings.TrimSpace(params.IMDBID); imdbID != "" {
		query.Set("imdb_id", imdbID)
	}
	if params.Limit > 0 {
		query.Set("limit", strconv.Itoa(params.Limit))
	}
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to http.NewRequestWithContext: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	res, err := s.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to http.Client.Do: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, invalidStatusError(res)
	}

	var subxResponse struct {
		Items []struct {
			ID           string `json:"id"`
			VideoType    string `json:"video_type"`
			Title        string `json:"title"`
			Season       int    `json:"season"`
			Episode      int    `json:"episode"`
			IMDBID       string `json:"imdb_id"`
			Description  string `json:"description"`
			UploaderName string `json:"uploader_name"`
			PostedAt     string `json:"posted_at"`
			Downloads    int    `json:"downloads"`
		} `json:"items"`
		Total int `json:"total"`
	}

	err = json.NewDecoder(res.Body).Decode(&subxResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to json.NewDecoder.Decode: %w", err)
	}

	subtitles := &Subtitles{
		TotalRecords: subxResponse.Total,
		Subtitles:    make([]*Subtitle, 0, len(subxResponse.Items)),
	}
	for _, item := range subxResponse.Items {
		subtitles.Subtitles = append(subtitles.Subtitles, &Subtitle{
			ID:               item.ID,
			VideoType:        item.VideoType,
			Title:            item.Title,
			Season:           item.Season,
			Episode:          item.Episode,
			IMDBID:           item.IMDBID,
			Description:      item.Description,
			UploaderName:     item.UploaderName,
			PostedAt:         item.PostedAt,
			Downloads:        item.Downloads,
			DescriptionWords: alphaNumericDistinctLowercaseWords(item.Title + " " + item.Description),
		})
	}

	return subtitles, nil
}

// DownloadSubtitle retrieves a specific subtitle file contents by its ID using the supplied token.
func (s *SubX) DownloadSubtitle(ctx context.Context, apiKey string, ID string) (*SubtitleContents, error) {
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("").Start(ctx, "subx.SubX.DownloadSubtitle")
	defer span.End()

	if apiKey == "" {
		return nil, fmt.Errorf("api key is empty")
	}

	if s.HttpClient == nil {
		return nil, fmt.Errorf("http client is nil")
	}

	endpoint := strings.TrimRight(s.BaseURL, "/") + "/api/subtitles/" + url.PathEscape(ID) + "/download"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to http.NewRequestWithContext: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	res, err := s.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to http.Client.Do: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, invalidStatusError(res)
	}

	filename := downloadFilename(res.Header.Get("Content-Disposition"))
	maxDownloadSize := maxSubtitleFileSize
	if isArchive(filename) {
		maxDownloadSize = maxSubtitleArchiveSize
	}

	data, err := io.ReadAll(LimitReader(res.Body, int64(maxDownloadSize), ErrReadBeyondLimit))
	if err != nil {
		return nil, fmt.Errorf("failed to io.ReadAll: %w", err)
	}
	if len(data) == 0 {
		return nil, errors.New("subtitle download is empty")
	}

	subtitle, err := extractSubtitle(data, filename)
	if err != nil {
		return nil, err
	}

	return subtitle, nil
}

func extractSubtitle(data []byte, filename string) (*SubtitleContents, error) {
	if len(data) == 0 {
		return nil, errors.New("subtitle download is empty")
	}

	if !isArchive(filename) {
		if len(data) > maxSubtitleFileSize {
			return nil, fmt.Errorf("subtitle file exceeds %d bytes: %w", maxSubtitleFileSize, ErrReadBeyondLimit)
		}
		return &SubtitleContents{
			Name: filename,
			Data: data,
		}, nil
	}

	archive, err := unarr.NewArchiveFromMemory(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unarr.NewArchiveFromMemory: %w", err)
	}
	defer archive.Close()

	for {
		err = archive.Entry()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to unarr.Archive.Entry: %w", err)
		}

		name := archive.Name()
		if !isSubtitle(name) {
			continue
		}

		if archive.Size() > maxSubtitleFileSize {
			return nil, fmt.Errorf("subtitle file exceeds %d bytes: %w", maxSubtitleFileSize, ErrReadBeyondLimit)
		}

		subtitleData, err := archive.ReadAll()
		if err != nil {
			return nil, fmt.Errorf("failed to unarr.Archive.ReadAll: %w", err)
		}
		if len(subtitleData) > maxSubtitleFileSize {
			return nil, fmt.Errorf("subtitle file exceeds %d bytes: %w", maxSubtitleFileSize, ErrReadBeyondLimit)
		}
		if len(subtitleData) == 0 {
			return nil, errors.New("subtitle is empty")
		}

		return &SubtitleContents{
			Name: path.Base(name),
			Data: subtitleData,
		}, nil
	}

	return nil, errors.New("no subtitle file found in archive")
}

func downloadFilename(contentDisposition string) string {
	filename := "subtitle.srt"
	if _, params, err := mime.ParseMediaType(contentDisposition); err == nil {
		if value := strings.TrimSpace(params["filename"]); value != "" {
			filename = value
		}
	}

	filename = strings.ReplaceAll(strings.TrimSpace(filename), "\\", "/")
	filename = path.Base(filename)
	if filename == "" || filename == "." || filename == "/" {
		return "subtitle.srt"
	}

	return filename
}

func invalidStatusError(res *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(res.Body, maxErrorBodySize))
	if len(body) == 0 {
		return fmt.Errorf("invalid status code: %d", res.StatusCode)
	}

	return fmt.Errorf("invalid status code: %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
}

func isSubtitle(filename string) bool {
	switch strings.ToLower(path.Ext(filename)) {
	case ".srt", ".sub", ".ssa", ".ass":
		return true
	default:
		return false
	}
}

func isArchive(filename string) bool {
	switch strings.ToLower(path.Ext(filename)) {
	case ".zip", ".rar", ".7z", ".tar":
		return true
	default:
		return false
	}
}

// alphaNumericDistinctLowercaseWords processes a string, extracts alphanumeric words, converts them to lowercase, and returns a slice of unique words in the order they appear.
func alphaNumericDistinctLowercaseWords(s string) []string {
	var result strings.Builder
	for i := 0; i < len(s); i++ {
		b := s[i]
		if ('a' <= b && b <= 'z') ||
			('A' <= b && b <= 'Z') ||
			('0' <= b && b <= '9') ||
			b == ' ' {
			result.WriteByte(byte(unicode.ToLower(rune(b))))
		} else {
			result.WriteByte(' ')
		}
	}
	m := make(map[string]struct{})
	fields := strings.Fields(result.String())
	j := 0
	for i := 0; i < len(fields); i++ {
		if _, ok := m[fields[i]]; !ok {
			m[fields[i]] = struct{}{}
			fields[j] = fields[i]
			j++
		}
	}
	return fields[:j]
}

// Score calculates a match score between a given string and the subtitle's description words.
func (f *Subtitle) Score(s string) int {
	var score int
	inputWords := alphaNumericDistinctLowercaseWords(s)
	for _, word := range inputWords {
		for _, descriptionWord := range f.DescriptionWords {
			if word == descriptionWord {
				score++
			}
		}
	}
	return score
}

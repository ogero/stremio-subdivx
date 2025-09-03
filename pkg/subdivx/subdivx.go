package subdivx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gen2brain/go-unarr"
	"github.com/ogero/stremio-subdivx/pkg/transport"
	"go.opentelemetry.io/otel/trace"
)

// Token represents the validation token and associated cookie for Subdivx API requests.
type Token struct {
	Token  string
	Cookie *http.Cookie
}

// Subtitles holds the total number of records and the corresponding IDs of a subset of them.
type Subtitles struct {
	TotalRecords int
	Subtitles    []*Subtitle
}
type Subtitle struct {
	ID               int
	Title            string
	Description      string
	DescriptionWords []string
}

// SubtitleContents holds content of a subtitle.
type SubtitleContents struct {
	Name string
	Data []byte
}

// Subdivx defines the methods to interact with the Subdivx service.
type Subdivx interface {
	// GetToken retrieves the validation token and associated cookie for Subdivx API requests.
	GetToken(ctx context.Context) (*Token, error)
	// GetSubtitles fetches subtitles for a given title.
	GetSubtitles(ctx context.Context, token *Token, title string) (*Subtitles, error)
	// GetSubtitle retrieves a specific subtitle file contents by its ID.
	GetSubtitle(ctx context.Context, ID string) (*SubtitleContents, error)
}

// NewSubdivx creates a new instance of the Subdivx service.
func NewSubdivx() Subdivx {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100

	rt := transport.NewModifyHeadersRoundTripper(t,
		transport.WithAcceptLanguage("es-AR,es;q=0.9,en;q=0.8"), // avoid IP-based language detection
		transport.WithUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36"),
	)

	return &subdivx{
		httpClient: &http.Client{
			Timeout:   time.Second * 10,
			Transport: rt,
		},
		versionREMatcher: regexp.MustCompile(`>v([0-9.a-z]+)<`),
		baseURL:          "https://www.subdivx.com",
	}
}

type subdivx struct {
	httpClient       *http.Client
	versionREMatcher *regexp.Regexp
	baseURL          string
}

// GetToken retrieves the validation token and associated cookie for Subdivx API requests.
func (s *subdivx) GetToken(ctx context.Context) (*Token, error) {

	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("").Start(ctx, "subdivx.Subdivx.GetToken")
	defer span.End()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/inc/gt.php?gt=1", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to http.NewRequestWithContext: %w", err)
	}

	res, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to http.Client.Get: %w", err)
	}
	defer res.Body.Close()

	tokenResponse := struct {
		Token string `json:"token"`
	}{}

	err = json.NewDecoder(res.Body).Decode(&tokenResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to json.NewDecoder.Decode: %w", err)
	}

	cookie, err := http.ParseSetCookie(res.Header.Get("Set-Cookie"))
	if err != nil {
		return nil, fmt.Errorf("failed to http.ParseSetCookie: %w", err)
	}

	return &Token{
		Token:  tokenResponse.Token,
		Cookie: cookie,
	}, nil
}

// GetSubtitles fetches subtitles for a given title.
func (s *subdivx) GetSubtitles(ctx context.Context, token *Token, title string) (*Subtitles, error) {

	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("").Start(ctx, "subdivx.Subdivx.GetSubtitles")
	defer span.End()

	webVersion, err := func() (string, error) {

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL, nil)
		if err != nil {
			return "", fmt.Errorf("failed to http.NewRequestWithContext: %w", err)
		}

		res, err := s.httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to http.Client.Get: %w", err)
		}
		defer res.Body.Close()

		html, err := io.ReadAll(res.Body)
		if err != nil {
			return "", fmt.Errorf("failed to io.ReadAll: %w", err)
		}

		matches := s.versionREMatcher.FindSubmatch(html)
		if matches == nil || len(matches) != 2 {
			return "", fmt.Errorf("failed to regexp.Regexp.FindSubmatch")
		}

		return string(bytes.ReplaceAll(matches[1], []byte("."), []byte(""))), nil
	}()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch web version: failed to io.ReadAll: %w", err)
	}

	formData := url.Values{}
	formData.Set("tabla", "resultados")
	formData.Set("filtros", "")
	formData.Set("buscar"+webVersion, title)
	formData.Set("token", token.Token)

	encodedForm := formData.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+`/inc/ajax.php`, strings.NewReader(encodedForm))
	if err != nil {
		return nil, fmt.Errorf("failed to http.NewRequestWithContext: %w", err)
	}

	req.Header.Set("Cookie", token.Cookie.String())
	req.Header.Set("Content-Type", `application/x-www-form-urlencoded; charset=UTF-8`)

	res, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to http.Client.Do: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("invalid status code: %d", res.StatusCode)
	}

	subdivxResponse := struct {
		ITotalRecords int `json:"iTotalRecords"`
		AaData        []struct {
			ID          int    `json:"id"`
			Title       string `json:"titulo"`
			Description string `json:"descripcion"`
		} `json:"aaData"`
	}{}

	err = json.NewDecoder(res.Body).Decode(&subdivxResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to json.NewDecoder.Decode: %w", err)
	}

	subtitles := &Subtitles{
		TotalRecords: subdivxResponse.ITotalRecords,
		Subtitles:    make([]*Subtitle, 0, subdivxResponse.ITotalRecords),
	}
	for _, aaData := range subdivxResponse.AaData {
		subtitles.Subtitles = append(subtitles.Subtitles, &Subtitle{
			ID:               aaData.ID,
			Title:            aaData.Title,
			Description:      aaData.Description,
			DescriptionWords: alphaNumericDistinctLowercaseWords(aaData.Title + " " + aaData.Description),
		})
	}

	return subtitles, nil
}

// GetSubtitle retrieves the subtitles archive for the specified ID, extracts it and returns the content of the first SRT file on it
func (s *subdivx) GetSubtitle(ctx context.Context, ID string) (*SubtitleContents, error) {

	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("").Start(ctx, "subdivx.Subdivx.GetSubtitle")
	defer span.End()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/descargar.php?id="+ID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to http.NewRequestWithContext: %w", err)
	}

	res, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to http.Client.Do: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status code: %d", res.StatusCode)
	}

	lr := LimitReader(res.Body, 500*1024, ErrReadBeyondLimit)

	archiveBytes, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("failed to io.ReadAll: %w", err)
	}

	if len(archiveBytes) == 0 {
		return nil, errors.New("archive is empty")
	}

	ar, err := unarr.NewArchiveFromMemory(archiveBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to unarr.NewArchiveFromReader: %w", err)
	}
	defer ar.Close()

	var sub *SubtitleContents
	for {
		err = ar.Entry()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, fmt.Errorf("failed to unarr.Archive.Entry: %w", err)
			}
		}
		fName := ar.Name()
		if isSubtitle(ar.Name()) {
			fData, err := ar.ReadAll()
			if err != nil {
				return nil, fmt.Errorf("failed to unarr.Archive.ReadAll: %w", err)
			}
			sub = &SubtitleContents{
				Name: fName,
				Data: fData,
			}
		}
	}

	if sub == nil {
		return nil, fmt.Errorf("no subtitle file found in archive")
	}

	return sub, nil
}

func isSubtitle(filename string) bool {
	lcFilename := strings.ToLower(filename)
	if len(lcFilename) > 4 {
		ext := lcFilename[len(lcFilename)-4:]
		switch ext {
		case ".srt", ".sub", ".ssa":
			return true
		}
	}
	return false
}

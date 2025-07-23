package imdb

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/StalkR/imdb"
	"github.com/ogero/stremio-subdivx/pkg/transport"
	"go.opentelemetry.io/otel/trace"
)

type stalkrIMDB struct {
	httpClient *http.Client
	getTitle   func(c *http.Client, id string) (*imdb.Title, error)
}

// NewStalkrIMDB creates a new instance of the Stalkr implementation of the IMDB service.
func NewStalkrIMDB() IMDB {

	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100

	rt := transport.NewModifyHeadersRoundTripper(t,
		transport.WithAcceptLanguage("en"), // avoid IP-based language detection
		transport.WithUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36"),
	)

	return &stalkrIMDB{
		httpClient: &http.Client{
			Timeout:   time.Second * 10,
			Transport: rt,
		},
		getTitle: imdb.NewTitle,
	}
}

// GetTitle gets a Title by its ID.
func (c *stalkrIMDB) GetTitle(ctx context.Context, imdbID string) (*Title, error) {

	_, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("").Start(ctx, "imdb.IMDB.GetTitle")
	defer span.End()

	imdbResult, err := c.getTitle(c.httpClient, imdbID)
	if err != nil {
		return nil, fmt.Errorf("failed to stalkrIMDB.getTitle: %w", err)
	}

	return &Title{
		Name: imdbResult.Name,
		Year: imdbResult.Year,
	}, nil
}

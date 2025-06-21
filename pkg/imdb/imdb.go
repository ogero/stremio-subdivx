package imdb

import (
	"fmt"
	"net/http"
	"time"

	"github.com/StalkR/imdb"
)

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36"

var httpClient = &http.Client{
	Timeout:   time.Second * 10,
	Transport: &customTransport{http.DefaultTransport},
}

type customTransport struct {
	http.RoundTripper
}

func (e *customTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	defer time.Sleep(time.Second)         // don't go too fast or risk being blocked by awswaf
	r.Header.Set("Accept-Language", "en") // avoid IP-based language detection
	r.Header.Set("User-Agent", userAgent)
	return e.RoundTripper.RoundTrip(r)
}

// FetchTitle gets, parses and returns a Title by its ID.
func FetchTitle(id string) (*imdb.Title, error) {

	imdbResults, err := imdb.NewTitle(httpClient, id)
	if err != nil {
		return nil, fmt.Errorf("failed to imdb.NewTitle: %w", err)
	}

	return imdbResults, nil
}

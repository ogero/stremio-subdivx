package loki

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Loki represents an interface for retrieving search and download statistics.
// GetSearches24 retrieves the total searches within the last 24 hours.
// GetDownloads24 retrieves the total downloads within the last 24 hours.
type Loki interface {
	// GetSearches24 retrieves the total number of searches performed in the last 24 hours. Returns the count or an error if retrieval fails.
	GetSearches24() (int, error)
	// GetDownloads24 retrieves the total number of downloads in the last 24 hours and returns an integer count and an error if applicable.
	GetDownloads24() (int, error)
}

type stremioSubdivxLoki struct {
	httpClient *http.Client
	lokiHost   string
}

func NewLoki(lokiHost string) Loki {
	return &stremioSubdivxLoki{
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
		lokiHost: lokiHost,
	}
}

// GetSearches24 retrieves the total number of searches performed in the last 24 hours. Returns the count or an error if retrieval fails.
func (s *stremioSubdivxLoki) GetSearches24() (int, error) {
	return s.countLokiLogs("SubtitlesHandler")
}

// GetDownloads24 retrieves the total number of downloads in the last 24 hours and returns an integer count and an error if applicable.
func (s *stremioSubdivxLoki) GetDownloads24() (int, error) {
	return s.countLokiLogs("SubdivxSubtitleHandler")
}

func (s *stremioSubdivxLoki) countLokiLogs(search string) (int, error) {
	url := s.lokiHost + "/loki/api/v1/query"
	query := fmt.Sprintf("count(rate({service_name=\"stremio-subdivx\"} |= `%s` [24h]))", search)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to http.NewRequest: %w", err)
	}

	q := req.URL.Query()
	q.Add("query", query)
	req.URL.RawQuery = q.Encode()

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var lokiResp LokiResponse
	if err := json.NewDecoder(resp.Body).Decode(&lokiResp); err != nil {
		return 0, fmt.Errorf("failed to json.Decoder.Decode: %w", err)
	}

	if lokiResp.Status != "success" {
		return 0, fmt.Errorf("loki response status: %s", lokiResp.Status)
	}

	if lokiResp.Data.ResultType != "vector" {
		return 0, fmt.Errorf("loki response data result type: %s", lokiResp.Data.ResultType)
	}

	if len(lokiResp.Data.Result) != 1 {
		return 0, fmt.Errorf("loki response data result length: %d", len(lokiResp.Data.Result))
	}

	if len(lokiResp.Data.Result[0].Value) != 2 {
		return 0, fmt.Errorf("loki response data result value length: %d", len(lokiResp.Data.Result[0].Value))
	}

	value, ok := (lokiResp.Data.Result[0].Value[1]).(string)
	if !ok {
		return 0, fmt.Errorf("failed to assert value to string: %v", value)
	}

	i, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("failed to strconv.Atoi: %w", err)
	}

	return i, nil
}

type LokiResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
			} `json:"metric"`
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

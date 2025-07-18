package imdb

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/StalkR/imdb"
)

func TestStalkrIMDB_GetTitle(t *testing.T) {

	s := &stalkrIMDB{
		httpClient: &http.Client{},
		getTitle: func(c *http.Client, id string) (*imdb.Title, error) {
			if id == "tt12345" {
				return &imdb.Title{
					Name: "mockTitle",
					Year: 2025,
				}, nil
			}
			return nil, fmt.Errorf("expected id tt12345, got %s", id)
		},
	}

	title, err := s.GetTitle(context.Background(), "tt12345")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if title.Name != "mockTitle" {
		t.Errorf("expected name 'mockTitle', got %v", title.Name)
	}
	if title.Year != 2025 {
		t.Errorf("expected year 2025, got %v", title.Year)
	}
}

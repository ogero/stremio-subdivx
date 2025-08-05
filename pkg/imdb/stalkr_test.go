package imdb

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/StalkR/imdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)

	assert.Equal(t, "mockTitle", title.Name)
	assert.Equal(t, 2025, title.Year)
}

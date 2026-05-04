package subx

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadSubtitleExtractsArchiveWithUnarr(t *testing.T) {
	archive := new(bytes.Buffer)
	zipWriter := zip.NewWriter(archive)
	file, err := zipWriter.Create("nested/subtitle.srt")
	require.NoError(t, err)
	_, err = io.WriteString(file, "Mock subtitle content")
	require.NoError(t, err)
	require.NoError(t, zipWriter.Close())

	subx := &SubX{
		HttpClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, "Bearer api-key", r.Header.Get("Authorization"))
				assert.Equal(t, "/api/subtitles/subtitle-id/download", r.URL.Path)

				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Disposition": []string{`attachment; filename="subtitle.zip"`},
					},
					Body: io.NopCloser(bytes.NewReader(archive.Bytes())),
				}, nil
			}),
		},
		BaseURL: "http://subx.test",
	}

	subtitle, err := subx.DownloadSubtitle(context.Background(), "api-key", "subtitle-id")
	require.NoError(t, err)

	assert.Equal(t, "subtitle.srt", subtitle.Name)
	assert.Equal(t, "Mock subtitle content", string(subtitle.Data))
}

func TestExtractSubtitleFallsBackToRawSubtitle(t *testing.T) {
	subtitle, err := extractSubtitle([]byte("Mock subtitle content"), "subtitle.srt")
	require.NoError(t, err)

	assert.Equal(t, "subtitle.srt", subtitle.Name)
	assert.Equal(t, "Mock subtitle content", string(subtitle.Data))
}

func TestDownloadSubtitleRejectsOversizedDownload(t *testing.T) {
	subx := &SubX{
		HttpClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Disposition": []string{`attachment; filename="subtitle.srt"`},
					},
					Body: io.NopCloser(strings.NewReader(strings.Repeat("x", maxSubtitleFileSize+1))),
				}, nil
			}),
		},
		BaseURL: "http://subx.test",
	}

	_, err := subx.DownloadSubtitle(context.Background(), "api-key", "subtitle-id")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrReadBeyondLimit), "expected ErrReadBeyondLimit, got %v", err)
}

func TestExtractSubtitleRejectsOversizedArchivedSubtitle(t *testing.T) {
	archive := new(bytes.Buffer)
	zipWriter := zip.NewWriter(archive)
	file, err := zipWriter.Create("subtitle.srt")
	require.NoError(t, err)
	_, err = io.Copy(file, strings.NewReader(strings.Repeat("x", maxSubtitleFileSize+1)))
	require.NoError(t, err)
	require.NoError(t, zipWriter.Close())

	_, err = extractSubtitle(archive.Bytes(), "subtitle.zip")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrReadBeyondLimit), "expected ErrReadBeyondLimit, got %v", err)
}

type roundTripFunc func(r *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

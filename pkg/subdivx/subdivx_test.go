package subdivx

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/inc/gt.php?gt=1" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Set-Cookie", "mockCookie=mockValue; Path=/; HttpOnly")
			json.NewEncoder(w).Encode(map[string]string{"token": "mockToken"})
		} else {
			t.Fatalf("unexpected request %v", r)
		}
	}))
	defer server.Close()

	s := &subdivx{
		httpClient:       &http.Client{},
		versionREMatcher: regexp.MustCompile(`>v([0-9.a-z]+)<`),
		baseURL:          server.URL,
	}

	token, err := s.GetToken(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "mockToken", token.Token)
	assert.Equal(t, "mockCookie", token.Cookie.Name)
	assert.Equal(t, "mockValue", token.Cookie.Value)
}

func TestGetSubtitles(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/" && r.Method == http.MethodGet {

			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<html>\n<v>v1a.0b</v>"))

		} else if r.RequestURI == "/inc/ajax.php" && r.Method == http.MethodPost {

			cookie, err := http.ParseSetCookie(r.Header.Get("Cookie"))
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if cookie.Name != "mockCookie" || cookie.Value != "mockValue" {
				t.Errorf("expected cookie 'mockCookie=mockValue', got %v", cookie)
			}

			if v := r.FormValue("tabla"); v != "resultados" {
				t.Errorf("expected FormValue tabla=resultados, got %v", v)
			}

			if v := r.FormValue("filtros"); v != "" {
				t.Errorf("expected FormValue filtros=, got %v", v)
			}

			if v := r.FormValue("buscar1a0b"); v != "testTitle" {
				t.Errorf("expected FormValue buscar1a0b=testTitle, got %v", v)
			}

			if v := r.FormValue("token"); v != "mockToken" {
				t.Errorf("expected FormValue token=mockToken, got %v", v)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"iTotalRecords": 1,
				"aaData": []map[string]any{
					{"id": 123, "titulo": "My mock movie", "descripcion": "A mock movie description!"},
				},
			})
		} else {
			t.Fatalf("unexpected request %v", r)
		}
	}))
	defer server.Close()

	s := &subdivx{
		httpClient:       &http.Client{},
		versionREMatcher: regexp.MustCompile(`>v([0-9.a-z]+)<`),
		baseURL:          server.URL,
	}

	token := &Token{
		Token:  "mockToken",
		Cookie: &http.Cookie{Name: "mockCookie", Value: "mockValue"},
	}

	subtitles, err := s.GetSubtitles(context.Background(), token, "testTitle")
	require.NoError(t, err)

	if assert.Len(t, subtitles.Subtitles, 1) {
		assert.EqualValues(t, 123, subtitles.Subtitles[0].ID)
		assert.Equal(t, "My mock movie", subtitles.Subtitles[0].Title)
		assert.ElementsMatch(t, strings.Fields("my a mock movie description"), subtitles.Subtitles[0].DescriptionWords)
	}
}

func TestGetSubtitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		zipWriter := zip.NewWriter(buf)
		srtFile, _ := zipWriter.Create("subtitle.srt")
		io.WriteString(srtFile, "Mock subtitle content")
		zipWriter.Close()

		w.Write(buf.Bytes())
	}))
	defer server.Close()

	s := &subdivx{
		httpClient:       &http.Client{},
		versionREMatcher: regexp.MustCompile(`>v([0-9.a-z]+)<`),
		baseURL:          server.URL,
	}

	subtitle, err := s.GetSubtitle(context.Background(), "123")
	require.NoError(t, err)

	if assert.Equal(t, "subtitle.srt", subtitle.Name) {
		assert.Equal(t, "Mock subtitle content", string(subtitle.Data))
	}

}

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
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if token.Token != "mockToken" {
		t.Errorf("expected token 'mockToken', got %v", token.Token)
	}
	if token.Cookie.Name != "mockCookie" || token.Cookie.Value != "mockValue" {
		t.Errorf("expected cookie 'mockCookie=mockValue', got %v", token.Cookie)
	}
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
				"aaData": []map[string]int{
					{"id": 123},
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
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if subtitles.TotalRecords != 1 {
		t.Errorf("expected 1 total record, got %v", subtitles.TotalRecords)
	}
	if len(subtitles.IDs) != 1 || subtitles.IDs[0] != 123 {
		t.Errorf("expected IDs [123], got %v", subtitles.IDs)
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
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedName := "subtitle.srt"
	if subtitle.Name != expectedName {
		t.Errorf("expected subtitle name %v, got %v", expectedName, subtitle.Name)
	}

	expectedContent := "Mock subtitle content"
	if !strings.Contains(string(subtitle.Data), expectedContent) {
		t.Errorf("expected subtitle content to contain %q, got %q", expectedContent, string(subtitle.Data))
	}
}

package transport_test

import (
	"net/http"
	"testing"

	"github.com/ogero/stremio-subdivx/pkg/transport"
)

type mockRoundTripper struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

func TestModifyHeadersRoundTripper(t *testing.T) {
	mockRT := &mockRoundTripper{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			if got := req.Header.Get("User-Agent"); got != "TestAgent" {
				t.Errorf("User-Agent = %q, want %q", got, "TestAgent")
			}

			if got := req.Header.Get("Accept-Language"); got != "en-US" {
				t.Errorf("Accept-Language = %q, want %q", got, "en-US")
			}
			return nil, nil
		},
	}

	rt := transport.NewModifyHeadersRoundTripper(mockRT,
		transport.WithUserAgent("TestAgent"),
		transport.WithAcceptLanguage("en-US"))

	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}

	_, _ = rt.RoundTrip(req)
}

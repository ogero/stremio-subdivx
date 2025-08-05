package transport_test

import (
	"net/http"
	"testing"

	"github.com/ogero/stremio-subdivx/pkg/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			assert.Equal(t, "TestAgent", req.Header.Get("User-Agent"))
			assert.Equal(t, "en-US", req.Header.Get("Accept-Language"))
			return nil, nil
		},
	}

	rt := transport.NewModifyHeadersRoundTripper(mockRT,
		transport.WithUserAgent("TestAgent"),
		transport.WithAcceptLanguage("en-US"))

	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)

	_, _ = rt.RoundTrip(req)
}

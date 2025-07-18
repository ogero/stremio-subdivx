package transport

import (
	"net/http"
)

// ModifyHeadersOption is a function type used to modify HTTP headers in a request.
// It takes a function that sets a header key and value, allowing for flexible header modification.
type ModifyHeadersOption func(func(key string, value string))

type modifyHeadersRoundTripper struct {
	roundTripper http.RoundTripper
	options      []ModifyHeadersOption
}

// NewModifyHeadersRoundTripper will add headers to a request.
func NewModifyHeadersRoundTripper(rt http.RoundTripper, opts ...ModifyHeadersOption) http.RoundTripper {
	return &modifyHeadersRoundTripper{roundTripper: rt, options: opts}
}

func (rt *modifyHeadersRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for _, opt := range rt.options {
		opt(req.Header.Set)
	}
	return rt.roundTripper.RoundTrip(req)
}

// WithUserAgent is a functional option to set the HTTP client user agent.
func WithUserAgent(userAgent string) ModifyHeadersOption {
	return func(f func(key string, value string)) {
		f("User-Agent", userAgent)
	}
}

// WithAcceptLanguage is a functional option to set the HTTP client accept language.
func WithAcceptLanguage(acceptLanguage string) ModifyHeadersOption {
	return func(f func(key string, value string)) {
		f("Accept-Language", acceptLanguage)
	}
}

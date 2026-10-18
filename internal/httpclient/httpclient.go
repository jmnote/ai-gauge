package httpclient

import (
	"net/http"
	"strings"
	"time"

	"github.com/jmnote/ai-gauge/internal/version"
)

var userAgent = "AI-Gauge/" + strings.TrimPrefix(version.Version, "v") + " (github.com/jmnote/ai-gauge)"

type userAgentTransport struct{ base http.RoundTripper }

func (t userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// A RoundTripper must not modify the caller's request.
	req = req.Clone(req.Context())
	req.Header.Set("User-Agent", userAgent)
	return t.base.RoundTrip(req)
}

var client = &http.Client{
	Transport: userAgentTransport{base: http.DefaultTransport},
	Timeout:   15 * time.Second,
}

// Client returns the HTTP client shared by every outbound request.
func Client() *http.Client { return client }

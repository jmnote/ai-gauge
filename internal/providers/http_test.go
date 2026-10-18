package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchAuthorizedJSONSendsUserAgent(t *testing.T) {
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	if _, err := fetchAuthorizedJSON(context.Background(), server.URL, "Test", map[string]string{"Authorization": "Bearer x"}); err != nil {
		t.Fatalf("fetchAuthorizedJSON: %v", err)
	}
	// The exact value is tested in internal/httpclient; this only checks that
	// provider requests go through the shared client.
	if !strings.HasPrefix(got, "AI-Gauge/") {
		t.Fatalf("User-Agent = %q, want AI-Gauge/...", got)
	}
}

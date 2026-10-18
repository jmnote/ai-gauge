package httpclient

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserAgent(t *testing.T) {
	// Test builds keep version.Version's "v0.0.0" default; the "v" is dropped.
	if want := "AI-Gauge/0.0.0 (github.com/jmnote/ai-gauge)"; userAgent != want {
		t.Fatalf("userAgent = %q, want %q", userAgent, want)
	}
}

func TestClientOverridesUserAgent(t *testing.T) {
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", "caller-set")
	resp, err := Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got != userAgent {
		t.Fatalf("User-Agent = %q, want %q", got, userAgent)
	}
	if req.Header.Get("User-Agent") != "caller-set" {
		t.Fatal("transport modified the caller's request")
	}
}

package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newMock returns a local OpenAI-compatible server returning a canned reply.
func newMock(t *testing.T, status int, content string, hdr *http.Header) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hdr != nil {
			*hdr = r.Header.Clone()
		}
		if status != http.StatusOK {
			w.WriteHeader(status)
			_, _ = w.Write([]byte("nope"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"choices":[{"message":{"content":%q}}]}`, content)
	}))
}

func clientFor(url string) *Client {
	return NewClient(&Config{BaseURL: url, APIKey: "sk-test-key", Model: "m"})
}

func TestChat_ParsesResponse(t *testing.T) {
	var hdr http.Header
	srv := newMock(t, http.StatusOK, "  hello  ", &hdr)
	defer srv.Close()

	out, err := clientFor(srv.URL).Chat(context.Background(), "sys", "usr")
	if err != nil {
		t.Fatal(err)
	}
	if out != "hello" {
		t.Errorf("out = %q, want hello (trimmed)", out)
	}
	if !strings.HasPrefix(hdr.Get("Authorization"), "Bearer ") {
		t.Errorf("missing bearer auth header: %q", hdr.Get("Authorization"))
	}
}

func TestChat_HTTPError(t *testing.T) {
	srv := newMock(t, http.StatusUnauthorized, "", nil)
	defer srv.Close()
	_, err := clientFor(srv.URL).Chat(context.Background(), "s", "u")
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("want HTTP 401 error, got %v", err)
	}
}

func TestChat_EmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer srv.Close()
	if _, err := clientFor(srv.URL).Chat(context.Background(), "s", "u"); err == nil {
		t.Fatal("expected empty-response error")
	}
}

func TestComplete(t *testing.T) {
	srv := newMock(t, http.StatusOK, "答案", nil)
	defer srv.Close()
	out, err := clientFor(srv.URL).Complete(context.Background(), "q")
	if err != nil || out != "答案" {
		t.Fatalf("Complete = %q, %v", out, err)
	}
}

func TestConfig_ConfiguredAndMask(t *testing.T) {
	var nilCfg *Config
	if nilCfg.Configured() {
		t.Error("nil config should not be configured")
	}
	c := &Config{BaseURL: "https://x/v1", APIKey: "sk-abcdefghij", Model: "m"}
	if !c.Configured() {
		t.Error("should be configured")
	}
	m := c.MaskedKey()
	if !strings.HasPrefix(m, "sk-a") || !strings.HasSuffix(m, "ghij") || !strings.Contains(m, "•") {
		t.Errorf("masked = %q", m)
	}
}

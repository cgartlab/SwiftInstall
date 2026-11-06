package advice

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cgartlab/SwiftInstall/internal/llm"
)

// TestExplain_EndToEnd exercises advice.Explain through a real llm.Client
// against a local mock of an OpenAI-compatible endpoint (no network / no key).
func TestExplain_EndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"choices":[{"message":{"content":"这是一个截图工具。免费平替：Capso。"}}]}`)
	}))
	defer srv.Close()

	c := llm.NewClient(&llm.Config{BaseURL: srv.URL, APIKey: "sk-test", Model: "m"})
	out, err := Explain(context.Background(), c, "CleanShot X")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Capso") {
		t.Errorf("unexpected answer: %q", out)
	}
}

package ui

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cgartlab/SwiftInstall/internal/engine"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
)

// JSONRenderer outputs results as structured JSON to stdout.
// Designed for piping to other tools (jq, scripts, etc.).
type JSONRenderer struct{}

func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{}
}

func (j *JSONRenderer) Header(total int, source string, action string) {
	// No header in JSON mode — all output is in Summary
}

func (j *JSONRenderer) Progress(index int, total int, pkg manifest.Package, result engine.Result) {
	// No incremental output in JSON mode — all output is in Summary
}

func (j *JSONRenderer) Summary(summary *engine.Summary) {
	if summary == nil {
		return
	}

	// Create a clean JSON structure
	type jsonResult struct {
		ID       string `json:"id"`
		Category string `json:"category,omitempty"`
		Status   string `json:"status"`
		Error    string `json:"error,omitempty"`
		Duration string `json:"duration,omitempty"`
	}

	type jsonSummary struct {
		Total     int          `json:"total"`
		Succeeded int          `json:"succeeded"`
		Skipped   int          `json:"skipped"`
		Failed    int          `json:"failed"`
		Duration  string       `json:"duration"`
		Results   []jsonResult `json:"results"`
	}

	out := jsonSummary{
		Total:     summary.Total,
		Succeeded: summary.Succeeded,
		Skipped:   summary.Skipped,
		Failed:    summary.Failed,
		Duration:  fmt.Sprintf("%.1fs", summary.Duration.Seconds()),
		Results:   make([]jsonResult, 0, len(summary.Results)),
	}

	for _, r := range summary.Results {
		out.Results = append(out.Results, jsonResult{
			ID:       r.Package.ID,
			Category: r.Package.Category,
			Status:   string(r.Status),
			Error:    r.Error,
			Duration: fmt.Sprintf("%.1fs", r.Duration.Seconds()),
		})
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to marshal JSON: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

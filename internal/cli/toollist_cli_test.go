package cli

import (
	"encoding/json"
	"io"
	"os"
	"testing"
)

func TestToollistListFormatJSON(t *testing.T) {
	cmd := newToollistListCmd()
	cmd.Flags().String("format", "", "")
	_ = cmd.Flags().Set("format", "json")

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	runErr := cmd.Execute()
	_ = w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)

	if runErr != nil {
		t.Fatalf("execute: %v", runErr)
	}
	var rows []map[string]any
	if err := json.Unmarshal(out, &rows); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, string(out))
	}
	if len(rows) == 0 {
		t.Error("expected at least one toollist row")
	}
	if _, ok := rows[0]["id"]; !ok {
		t.Errorf("row missing id field: %v", rows[0])
	}
}

func TestToollistListFormatSilent(t *testing.T) {
	cmd := newToollistListCmd()
	cmd.Flags().String("format", "", "")
	_ = cmd.Flags().Set("format", "silent")
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	_ = cmd.Execute()
	_ = w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	if len(out) != 0 {
		t.Errorf("silent should print nothing, got %d bytes", len(out))
	}
}

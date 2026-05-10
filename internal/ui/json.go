package ui

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cgartlab/SwiftInstall/internal/engine"
	"github.com/cgartlab/SwiftInstall/internal/manifest"
)

type JSONRenderer struct{}

func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{}
}

func (j *JSONRenderer) Start(total int, manifestPath string, action string) {}

func (j *JSONRenderer) Progress(pkg manifest.Package, status string, err error) {}

func (j *JSONRenderer) Done(summary *engine.Summary) {
	if summary == nil {
		return
	}
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal summary: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

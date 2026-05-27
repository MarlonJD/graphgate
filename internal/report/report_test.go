package report

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/MarlonJD/graphgate/internal/config"
	"github.com/MarlonJD/graphgate/internal/core"
)

func TestRenderJSON(t *testing.T) {
	summary := sampleSummary()
	data, err := RenderJSON(summary)
	if err != nil {
		t.Fatalf("RenderJSON() error = %v", err)
	}

	var decoded Summary
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !decoded.OK || decoded.Operations[0].Name != "GetViewer" {
		t.Fatalf("decoded summary = %#v", decoded)
	}
}

func TestRenderMarkdown(t *testing.T) {
	markdown := string(RenderMarkdown(sampleSummary()))
	for _, want := range []string{"# GraphGate Report", "Status: **PASS**", "GetViewer", "graphgate.manifest.json"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("markdown missing %q:\n%s", want, markdown)
		}
	}
}

func sampleSummary() Summary {
	result := core.ValidationResult{
		Schema:         "schema.graphql",
		OperationFiles: []string{"operations/GetViewer.graphql"},
		Operations: []core.Operation{{
			Name:       "GetViewer",
			ID:         "0123456789abcdef",
			SHA256:     "0123456789abcdef",
			File:       "operations/GetViewer.graphql",
			Normalized: "query GetViewer{viewer}",
		}},
	}
	manifest := core.BuildManifest(&config.Config{
		Manifest: config.ManifestConfig{Format: config.DefaultManifestFormat},
	}, result)
	return NewSummary(result, manifest, "graphgate.manifest.json")
}

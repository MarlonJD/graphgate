package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildManifestIsDeterministic(t *testing.T) {
	cfg := loadTestConfig(t, map[string]string{
		"schema.graphql":       "type Query { a: String!, b: String! }\n",
		"operations/b.graphql": "query B { b }\n",
		"operations/a.graphql": "query A { a }\n",
	})
	result, err := ValidateProject(cfg)
	if err != nil {
		t.Fatalf("ValidateProject() error = %v", err)
	}
	if !result.OK() {
		t.Fatalf("issues = %#v", result.Issues)
	}

	manifest := BuildManifest(cfg, result)
	if len(manifest.Operations) != 2 {
		t.Fatalf("operations len = %d", len(manifest.Operations))
	}
	if manifest.Operations[0].Name != "A" || manifest.Operations[1].Name != "B" {
		t.Fatalf("operations not sorted: %#v", manifest.Operations)
	}

	left, err := ManifestBytes(manifest)
	if err != nil {
		t.Fatal(err)
	}
	right, err := ManifestBytes(BuildManifest(cfg, result))
	if err != nil {
		t.Fatal(err)
	}
	if string(left) != string(right) {
		t.Fatalf("manifest bytes differ")
	}
}

func TestCheckManifestPassAndFail(t *testing.T) {
	cfg := loadTestConfig(t, map[string]string{
		"schema.graphql":               "type Query { viewer: String! }\n",
		"operations/GetViewer.graphql": "query GetViewer { viewer }\n",
	})
	result, err := ValidateProject(cfg)
	if err != nil {
		t.Fatalf("ValidateProject() error = %v", err)
	}
	manifest := BuildManifest(cfg, result)
	path := filepath.Join(t.TempDir(), "graphgate.manifest.json")

	if err := WriteManifest(path, manifest); err != nil {
		t.Fatal(err)
	}
	matches, err := CheckManifest(path, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if !matches {
		t.Fatalf("CheckManifest() = false, want true")
	}

	if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	matches, err = CheckManifest(path, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if matches {
		t.Fatalf("CheckManifest() = true, want false")
	}
}

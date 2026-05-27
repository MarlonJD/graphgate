package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	dir := newProject(t)
	writeFile(t, dir, "schema.graphql", "type Query { viewer: String! }\n")
	writeFile(t, dir, "operations/GetViewer.graphql", "query GetViewer { viewer }\n")
	writeConfig(t, dir, "./operations/**/*.graphql")

	cfg, err := Load(filepath.Join(dir, DefaultConfigPath))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Schema != "./schema.graphql" {
		t.Fatalf("Schema = %q", cfg.Schema)
	}
	files, err := cfg.OperationFiles()
	if err != nil {
		t.Fatalf("OperationFiles() error = %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("OperationFiles() len = %d", len(files))
	}
}

func TestLoadMissingSchema(t *testing.T) {
	dir := newProject(t)
	writeFile(t, dir, "operations/GetViewer.graphql", "query GetViewer { viewer }\n")
	writeConfig(t, dir, "./operations/**/*.graphql")

	_, err := Load(filepath.Join(dir, DefaultConfigPath))
	if err == nil || !strings.Contains(err.Error(), "schema file") {
		t.Fatalf("Load() error = %v, want missing schema", err)
	}
}

func TestLoadMissingOperations(t *testing.T) {
	dir := newProject(t)
	writeFile(t, dir, "schema.graphql", "type Query { viewer: String! }\n")
	writeConfig(t, dir, "./operations/**/*.graphql")

	_, err := Load(filepath.Join(dir, DefaultConfigPath))
	if err == nil || !strings.Contains(err.Error(), "matched no files") {
		t.Fatalf("Load() error = %v, want missing operations", err)
	}
}

func TestLoadBadRecursiveGlob(t *testing.T) {
	dir := newProject(t)
	writeFile(t, dir, "schema.graphql", "type Query { viewer: String! }\n")
	writeConfig(t, dir, "./operations/**/[broken.graphql")

	_, err := Load(filepath.Join(dir, DefaultConfigPath))
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("Load() error = %v, want invalid glob", err)
	}
}

func newProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "operations"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeConfig(t *testing.T, dir string, operationPattern string) {
	t.Helper()
	data := []byte("schema: ./schema.graphql\noperations:\n  - " + operationPattern + "\nmanifest:\n  format: graphgate\n  output: ./graphgate.manifest.json\n")
	if err := os.WriteFile(filepath.Join(dir, DefaultConfigPath), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

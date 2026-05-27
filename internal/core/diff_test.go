package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiffSchemasRemovedFieldImpactsOperation(t *testing.T) {
	dir := t.TempDir()
	writeProjectFile(t, dir, "base.graphql", "type Query { viewer: User! }\ntype User { id: ID!, name: String! }\n")
	cfg := loadTestConfig(t, map[string]string{
		"schema.graphql":               "type Query { viewer: User! }\ntype User { id: ID! }\n",
		"operations/GetViewer.graphql": "query GetViewer { viewer { id name } }\n",
	})
	basePath := filepath.Join(dir, "base.graphql")

	result, err := DiffSchemas(cfg, basePath)
	if err != nil {
		t.Fatalf("DiffSchemas() error = %v", err)
	}
	if result.OK {
		t.Fatalf("DiffSchemas().OK = true, want false")
	}
	if len(result.ImpactedOperations) != 1 {
		t.Fatalf("impacted operations = %#v", result.ImpactedOperations)
	}
	if result.ImpactedOperations[0].Operation != "GetViewer" {
		t.Fatalf("impact = %#v", result.ImpactedOperations[0])
	}
}

func TestDiffSchemasUnrelatedAdditionPasses(t *testing.T) {
	dir := t.TempDir()
	writeProjectFile(t, dir, "base.graphql", "type Query { viewer: User! }\ntype User { id: ID! }\n")
	cfg := loadTestConfig(t, map[string]string{
		"schema.graphql":               "type Query { viewer: User!, version: String! }\ntype User { id: ID! }\n",
		"operations/GetViewer.graphql": "query GetViewer { viewer { id } }\n",
	})
	basePath := filepath.Join(dir, "base.graphql")

	result, err := DiffSchemas(cfg, basePath)
	if err != nil {
		t.Fatalf("DiffSchemas() error = %v", err)
	}
	if !result.OK {
		t.Fatalf("DiffSchemas().OK = false, changes=%#v impacts=%#v", result.BreakingChanges, result.ImpactedOperations)
	}
}

func writeProjectFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

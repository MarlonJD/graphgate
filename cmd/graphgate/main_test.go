package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunValidateInvalidOperationExitCode(t *testing.T) {
	dir := newCLIProject(t, map[string]string{
		"schema.graphql":               "type Query { viewer: String! }\n",
		"operations/GetViewer.graphql": "query GetViewer { missing }\n",
	})

	var stdout, stderr bytes.Buffer
	code := run([]string{"graphgate", "validate", "--config", filepath.Join(dir, "graphgate.yaml")}, &stdout, &stderr)
	if code != exitInvalidOperation {
		t.Fatalf("exit code = %d, want %d\nstderr=%s", code, exitInvalidOperation, stderr.String())
	}
}

func TestRunManifestCheckMismatchExitCode(t *testing.T) {
	dir := newCLIProject(t, map[string]string{
		"schema.graphql":               "type Query { viewer: String! }\n",
		"operations/GetViewer.graphql": "query GetViewer { viewer }\n",
		"graphgate.manifest.json":      "{}\n",
	})

	var stdout, stderr bytes.Buffer
	code := run([]string{"graphgate", "manifest", "--config", filepath.Join(dir, "graphgate.yaml"), "--check"}, &stdout, &stderr)
	if code != exitManifestMismatch {
		t.Fatalf("exit code = %d, want %d\nstderr=%s", code, exitManifestMismatch, stderr.String())
	}
}

func TestRunSmokeUsesContractTestCommand(t *testing.T) {
	dir := newCLIProject(t, map[string]string{
		"schema.graphql":                     "type Query { viewer: String! }\n",
		"operations/GetViewer.graphql":       "query GetViewer { viewer }\n",
		"graphgate/fixtures/get_viewer.json": `{"operation":"GetViewer","expectedStatus":200}`,
	})

	var stdout, stderr bytes.Buffer
	code := run([]string{"graphgate", "smoke", "--config", filepath.Join(dir, "graphgate.yaml"), "--timeout", "1ms"}, &stdout, &stderr)
	if code != exitTestFailure {
		t.Fatalf("exit code = %d, want %d\nstdout=%s\nstderr=%s", code, exitTestFailure, stdout.String(), stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("GraphGate Test Report")) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestRunSmokeSupportsSuiteTagExcludeAndEvidence(t *testing.T) {
	dir := newCLIProject(t, map[string]string{
		"schema.graphql":                         "type Query { viewer: String! }\n",
		"operations/GetViewer.graphql":           "query GetViewer { viewer }\n",
		"graphgate/fixtures/get_viewer.json":     `{"operation":"GetViewer","tags":["smoke","safe"],"expectedStatus":200}`,
		"graphgate/fixtures/destructive.json":    `{"operation":"GetViewer","tags":["smoke","destructive"],"expectedStatus":200}`,
		"graphgate/fixtures/reference_only.json": `{"operation":"GetViewer","tags":["reference"],"expectedStatus":200}`,
	})
	cfg := []byte("schema: ./schema.graphql\noperations:\n  - ./operations/**/*.graphql\nmanifest:\n  format: graphgate\n  output: ./graphgate.manifest.json\nenvironments:\n  local:\n    endpoint: http://127.0.0.1:1/graphql\ntests:\n  fixtures: ./graphgate/fixtures/**/*.json\n  suites:\n    safe-smoke:\n      tags: [smoke]\n      exclude: [destructive]\n")
	if err := os.WriteFile(filepath.Join(dir, "graphgate.yaml"), cfg, 0o644); err != nil {
		t.Fatal(err)
	}

	evidencePath := filepath.Join(dir, "graphgate", "reports", "evidence.json")
	var stdout, stderr bytes.Buffer
	code := run([]string{"graphgate", "smoke", "--config", filepath.Join(dir, "graphgate.yaml"), "--suite", "safe-smoke", "--tag", "safe", "--exclude", "reference", "--timeout", "1ms", "--evidence", evidencePath}, &stdout, &stderr)
	if code != exitTestFailure {
		t.Fatalf("exit code = %d, want %d\nstdout=%s\nstderr=%s", code, exitTestFailure, stdout.String(), stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("get_viewer.json")) {
		t.Fatalf("stdout = %s, want selected fixture", stdout.String())
	}
	if bytes.Contains(stdout.Bytes(), []byte("destructive.json")) || bytes.Contains(stdout.Bytes(), []byte("reference_only.json")) {
		t.Fatalf("stdout = %s, want excluded fixtures omitted", stdout.String())
	}
	data, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatalf("read evidence: %v", err)
	}
	if !bytes.Contains(data, []byte(`"suite": "safe-smoke"`)) || !bytes.Contains(data, []byte(`"fixtureCount": 1`)) {
		t.Fatalf("evidence = %s", string(data))
	}
}

func TestRunInitCreatesConfigAndDirectories(t *testing.T) {
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	var stdout, stderr bytes.Buffer
	code := run([]string{"graphgate", "init"}, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d\nstderr=%s", code, exitOK, stderr.String())
	}
	for _, path := range []string{"graphgate.yaml", "operations", "graphgate/fixtures", "graphgate/reports"} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Fatalf("%s not created: %v", path, err)
		}
	}
}

func newCLIProject(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, data := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg := []byte("schema: ./schema.graphql\noperations:\n  - ./operations/**/*.graphql\nmanifest:\n  format: graphgate\n  output: ./graphgate.manifest.json\nenvironments:\n  local:\n    endpoint: http://127.0.0.1:1/graphql\ntests:\n  fixtures: ./graphgate/fixtures/**/*.json\n")
	if err := os.WriteFile(filepath.Join(dir, "graphgate.yaml"), cfg, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

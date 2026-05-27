package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MarlonJD/graphgate/internal/config"
)

func TestRunContractTestsPassesFixtureAssertions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"viewer":{"id":"u1","name":"Burak"}}}`))
	}))
	defer server.Close()

	cfg := contractTestConfig(t, server.URL, `{
  "operation": "GetViewer",
  "variables": {},
  "expectedStatus": 200,
  "assertions": [
    {"path": "data.viewer.id", "type": "string"},
    {"path": "data.viewer.name", "equals": "Burak"}
  ]
}`)

	result, err := RunContractTests(context.Background(), cfg, TestOptions{Environment: "local", Timeout: time.Second})
	if err != nil {
		t.Fatalf("RunContractTests() error = %v", err)
	}
	if !result.OK || result.Passed != 1 || result.Failed != 0 {
		t.Fatalf("result = %#v", result)
	}
}

func TestRunContractTestsCapturesGraphQLErrorCodes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errors":[{"message":"nope","extensions":{"code":"FORBIDDEN"}}]}`))
	}))
	defer server.Close()

	cfg := contractTestConfig(t, server.URL, `{
  "operation": "GetViewer",
  "expectedStatus": 200,
  "expectedErrorCodes": ["FORBIDDEN"]
}`)

	result, err := RunContractTests(context.Background(), cfg, TestOptions{Environment: "local", Timeout: time.Second})
	if err != nil {
		t.Fatalf("RunContractTests() error = %v", err)
	}
	if !result.OK {
		t.Fatalf("result = %#v", result)
	}
}

func TestRunContractTestsUpdateSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"viewer":{"id":"u1"}}}`))
	}))
	defer server.Close()

	cfg := contractTestConfig(t, server.URL, `{"operation":"GetViewer"}`)
	result, err := RunContractTests(context.Background(), cfg, TestOptions{Environment: "local", Update: true, Timeout: time.Second})
	if err != nil {
		t.Fatalf("RunContractTests() error = %v", err)
	}
	if !result.OK || !result.Results[0].SnapshotUpdate {
		t.Fatalf("result = %#v", result)
	}

	fixtures, err := cfg.FixtureFiles()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(fixtures[0])
	if err != nil {
		t.Fatal(err)
	}
	var updated Fixture
	if err := json.Unmarshal(data, &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Snapshot == nil {
		t.Fatalf("snapshot was not written")
	}
}

func contractTestConfig(t *testing.T, endpoint string, fixture string) *config.Config {
	t.Helper()
	dir := t.TempDir()
	writeProjectFile(t, dir, "schema.graphql", "type Query { viewer: User! }\ntype User { id: ID!, name: String }\n")
	writeProjectFile(t, dir, "operations/GetViewer.graphql", "query GetViewer { viewer { id name } }\n")
	writeProjectFile(t, dir, "graphgate/fixtures/get_viewer.json", fixture)
	cfgData := "schema: ./schema.graphql\noperations:\n  - ./operations/**/*.graphql\nmanifest:\n  format: graphgate\n  output: ./graphgate.manifest.json\nenvironments:\n  local:\n    endpoint: " + endpoint + "\n    headers:\n      Authorization: Bearer ${GRAPHGATE_TOKEN}\ntests:\n  fixtures: ./graphgate/fixtures/**/*.json\nreports:\n  output: ./graphgate/reports\n"
	cfgPath := filepath.Join(dir, config.DefaultConfigPath)
	if err := os.WriteFile(cfgPath, []byte(cfgData), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	return cfg
}

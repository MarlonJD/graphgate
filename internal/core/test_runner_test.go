package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestRunContractTestsSupportsPersistedOnlyRuntimeOperationFixtures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-User-ID"); got != "42" {
			t.Fatalf("X-User-ID = %q, want 42", got)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if _, ok := payload["query"]; ok {
			t.Fatalf("query should be omitted for persisted-only fixture: %#v", payload)
		}
		if got := payload["operationName"]; got != "RuntimeViewer" {
			t.Fatalf("operationName = %v, want RuntimeViewer", got)
		}
		variables, ok := payload["variables"].(map[string]any)
		if !ok {
			t.Fatalf("variables = %#v, want object", payload["variables"])
		}
		if got := variables["searchText"]; got != "Ada" {
			t.Fatalf("searchText = %v, want Ada", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"viewer":{"friends":[{"id":"u2"}]}}}`))
	}))
	defer server.Close()

	t.Setenv("GRAPHGATE_SEARCH_TEXT", "Ada")
	cfg := contractTestConfig(t, server.URL, `{
  "operation": "GetViewer",
  "requestOperationName": "RuntimeViewer",
  "includeQuery": false,
  "headers": {
    "X-User-ID": "42"
  },
  "variables": {
    "searchText": "${GRAPHGATE_SEARCH_TEXT}"
  },
  "expectedStatus": 200,
  "assertions": [
    {"path": "data.viewer.friends", "type": "array", "minItems": 1}
  ]
}`)

	result, err := RunContractTests(context.Background(), cfg, TestOptions{Environment: "local", Timeout: time.Second})
	if err != nil {
		t.Fatalf("RunContractTests() error = %v", err)
	}
	if !result.OK || result.Passed != 1 || result.Failed != 0 {
		t.Fatalf("result = %#v", result)
	}
	if result.Results[0].RequestOperationName != "RuntimeViewer" {
		t.Fatalf("request operation = %q", result.Results[0].RequestOperationName)
	}
}

func TestRunContractTestsRequiresFixtureOperationCoverage(t *testing.T) {
	dir := t.TempDir()
	writeProjectFile(t, dir, "schema.graphql", "type Query { viewer: String!, cities: [String!]! }\n")
	writeProjectFile(t, dir, "operations/GetViewer.graphql", "query GetViewer { viewer }\n")
	writeProjectFile(t, dir, "operations/GetCities.graphql", "query GetCities { cities }\n")
	writeProjectFile(t, dir, "graphgate/fixtures/get_viewer.json", `{"operation":"GetViewer"}`)
	cfgData := "schema: ./schema.graphql\noperations:\n  - ./operations/**/*.graphql\nmanifest:\n  format: graphgate\n  output: ./graphgate.manifest.json\nenvironments:\n  local:\n    endpoint: http://127.0.0.1:1/graphql\ntests:\n  fixtures: ./graphgate/fixtures/**/*.json\n  requireOperationCoverage: true\n"
	cfgPath := filepath.Join(dir, config.DefaultConfigPath)
	if err := os.WriteFile(cfgPath, []byte(cfgData), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}

	_, err = RunContractTests(context.Background(), cfg, TestOptions{Environment: "local", Timeout: time.Second})
	if err == nil || !strings.Contains(err.Error(), "GetCities") {
		t.Fatalf("RunContractTests() error = %v, want missing GetCities coverage", err)
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

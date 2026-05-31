package core

import (
	"context"
	"encoding/json"
	"errors"
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

	result, err := RunContractTests(context.Background(), cfg, TestOptions{Environment: "local", Timeout: time.Second})
	if err != nil {
		t.Fatalf("RunContractTests() error = %v", err)
	}
	if result.OK || result.FailureClass != FailureClassCoverage || !strings.Contains(strings.Join(result.Failures, ","), "GetCities") {
		t.Fatalf("result = %#v, want missing GetCities coverage", result)
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

func TestRunContractTestsSelectsSuiteAndMergesProfiles(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want Bearer test-token", got)
		}
		if got := r.Header.Get("X-User-ID"); got != "u-member" {
			t.Fatalf("X-User-ID = %q, want u-member", got)
		}
		if got := r.Header.Get("X-Merge"); got != "fixture" {
			t.Fatalf("X-Merge = %q, want fixture", got)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		variables := payload["variables"].(map[string]any)
		if got := variables["searchText"]; got != "Ada" {
			t.Fatalf("searchText = %v, want Ada", got)
		}
		_, _ = w.Write([]byte(`{"data":{"viewer":{"id":"u1","name":"Ada"}}}`))
	}))
	defer server.Close()

	t.Setenv("GRAPHGATE_TOKEN", "test-token")
	t.Setenv("GRAPHGATE_USER_ID", "u-member")
	t.Setenv("GRAPHGATE_SEARCH_TEXT", "Ada")

	dir := t.TempDir()
	writeProjectFile(t, dir, "schema.graphql", "type Query { viewer: User! }\ntype User { id: ID!, name: String }\n")
	writeProjectFile(t, dir, "operations/GetViewer.graphql", "query GetViewer { viewer { id name } }\n")
	writeProjectFile(t, dir, "graphgate/fixtures/member.json", `{
  "operation": "GetViewer",
  "tags": ["smoke", "member"],
  "profile": "member",
  "headers": { "X-Merge": "fixture" },
  "assertions": [{ "path": "data.viewer.id", "type": "string" }]
}`)
	writeProjectFile(t, dir, "graphgate/fixtures/destructive.json", `{
  "operation": "GetViewer",
  "tags": ["smoke", "destructive"],
  "profile": "member"
}`)
	cfgData := "schema: ./schema.graphql\noperations:\n  - ./operations/**/*.graphql\nmanifest:\n  format: graphgate\n  output: ./graphgate.manifest.json\nenvironments:\n  local:\n    endpoint: " + server.URL + "\n    headers:\n      Authorization: Bearer ${GRAPHGATE_TOKEN}\n      X-Merge: env\nprofiles:\n  member:\n    headers:\n      X-User-ID: ${GRAPHGATE_USER_ID}\n      X-Merge: profile\n    variables:\n      searchText: ${GRAPHGATE_SEARCH_TEXT}\ntests:\n  fixtures: ./graphgate/fixtures/**/*.json\n  suites:\n    safe-smoke:\n      tags: [smoke]\n      exclude: [destructive]\n"
	cfgPath := filepath.Join(dir, config.DefaultConfigPath)
	if err := os.WriteFile(cfgPath, []byte(cfgData), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}

	result, err := RunContractTests(context.Background(), cfg, TestOptions{Environment: "local", Suite: "safe-smoke", Timeout: time.Second})
	if err != nil {
		t.Fatalf("RunContractTests() error = %v", err)
	}
	if !result.OK || result.Passed != 1 || requests != 1 {
		t.Fatalf("result = %#v requests=%d", result, requests)
	}
	if got := result.Selection.FixtureCount; got != 1 {
		t.Fatalf("selected fixtures = %d, want 1", got)
	}
}

func TestRunContractTestsSelectionErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"viewer":{"id":"u1"}}}`))
	}))
	defer server.Close()
	cfg := contractTestConfig(t, server.URL, `{"operation":"GetViewer","tags":["smoke"]}`)

	if _, err := RunContractTests(context.Background(), cfg, TestOptions{Environment: "local", Suite: "missing", Timeout: time.Second}); !errors.Is(err, ErrUnknownSuite) {
		t.Fatalf("unknown suite error = %v, want ErrUnknownSuite", err)
	}
	if _, err := RunContractTests(context.Background(), cfg, TestOptions{Environment: "local", Tags: []string{"missing"}, Timeout: time.Second}); !errors.Is(err, ErrNoSelectedFixtures) {
		t.Fatalf("empty selection error = %v, want ErrNoSelectedFixtures", err)
	}
}

func TestRunContractTestsReadinessRequiredEnvFailsBeforeRequests(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = w.Write([]byte(`{"data":{"viewer":{"id":"u1"}}}`))
	}))
	defer server.Close()

	dir := t.TempDir()
	writeProjectFile(t, dir, "schema.graphql", "type Query { viewer: User! }\ntype User { id: ID! }\n")
	writeProjectFile(t, dir, "operations/GetViewer.graphql", "query GetViewer { viewer { id } }\n")
	writeProjectFile(t, dir, "graphgate/fixtures/get_viewer.json", `{"operation":"GetViewer"}`)
	cfgData := "schema: ./schema.graphql\noperations:\n  - ./operations/**/*.graphql\nmanifest:\n  format: graphgate\n  output: ./graphgate.manifest.json\nenvironments:\n  local:\n    endpoint: " + server.URL + "\n    requiredEnv: [GRAPHGATE_REQUIRED_TEST_TOKEN]\ntests:\n  fixtures: ./graphgate/fixtures/**/*.json\n"
	cfgPath := filepath.Join(dir, config.DefaultConfigPath)
	if err := os.WriteFile(cfgPath, []byte(cfgData), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}

	result, err := RunContractTests(context.Background(), cfg, TestOptions{Environment: "local", Timeout: time.Second})
	if err != nil {
		t.Fatalf("RunContractTests() error = %v", err)
	}
	if result.OK || result.FailureClass != FailureClassReadiness || requests != 0 {
		t.Fatalf("result = %#v requests=%d", result, requests)
	}
}

func TestRunContractTestsRetriesConfiguredFailureClasses(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"errors":[{"message":"try again"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":{"viewer":{"id":"u1"}}}`))
	}))
	defer server.Close()

	dir := t.TempDir()
	writeProjectFile(t, dir, "schema.graphql", "type Query { viewer: User! }\ntype User { id: ID! }\n")
	writeProjectFile(t, dir, "operations/GetViewer.graphql", "query GetViewer { viewer { id } }\n")
	writeProjectFile(t, dir, "graphgate/fixtures/get_viewer.json", `{"operation":"GetViewer","expectedStatus":200}`)
	cfgData := "schema: ./schema.graphql\noperations:\n  - ./operations/**/*.graphql\nmanifest:\n  format: graphgate\n  output: ./graphgate.manifest.json\nenvironments:\n  local:\n    endpoint: " + server.URL + "\ntests:\n  fixtures: ./graphgate/fixtures/**/*.json\n  retry:\n    maxAttempts: 2\n    retryableFailureClasses: [status_mismatch]\n"
	cfgPath := filepath.Join(dir, config.DefaultConfigPath)
	if err := os.WriteFile(cfgPath, []byte(cfgData), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}

	result, err := RunContractTests(context.Background(), cfg, TestOptions{Environment: "local", Timeout: time.Second})
	if err != nil {
		t.Fatalf("RunContractTests() error = %v", err)
	}
	if !result.OK || requests != 2 || result.Results[0].Attempts != 2 {
		t.Fatalf("result = %#v requests=%d", result, requests)
	}
}

func TestRunContractTestsAllowsExplicitNegativeSelectionWithPositiveCoverageGate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errors":[{"message":"forbidden","extensions":{"code":"FORBIDDEN"}}]}`))
	}))
	defer server.Close()

	dir := t.TempDir()
	writeProjectFile(t, dir, "schema.graphql", "type Query { viewer: User! }\ntype User { id: ID! }\n")
	writeProjectFile(t, dir, "operations/GetViewer.graphql", "query GetViewer { viewer { id } }\n")
	writeProjectFile(t, dir, "graphgate/fixtures/positive.json", `{"operation":"GetViewer","tags":["positive"]}`)
	writeProjectFile(t, dir, "graphgate/fixtures/negative.json", `{"operation":"GetViewer","tags":["negative","security"],"expectedErrorCodes":["FORBIDDEN"]}`)
	cfgData := "schema: ./schema.graphql\noperations:\n  - ./operations/**/*.graphql\nmanifest:\n  format: graphgate\n  output: ./graphgate.manifest.json\nenvironments:\n  local:\n    endpoint: " + server.URL + "\ntests:\n  fixtures: ./graphgate/fixtures/**/*.json\n  coverage:\n    requirePositiveFixture: true\n"
	cfgPath := filepath.Join(dir, config.DefaultConfigPath)
	if err := os.WriteFile(cfgPath, []byte(cfgData), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}

	result, err := RunContractTests(context.Background(), cfg, TestOptions{Environment: "local", Tags: []string{"negative", "security"}, Timeout: time.Second})
	if err != nil {
		t.Fatalf("RunContractTests() error = %v", err)
	}
	if !result.OK || result.Passed != 1 || result.FailureClass == FailureClassCoverage {
		t.Fatalf("result = %#v", result)
	}
}

func TestEvaluateAssertionSupportsWildcardAndArrayOperators(t *testing.T) {
	decoded := map[string]any{
		"data": map[string]any{
			"items": []any{
				map[string]any{"id": "a1", "score": float64(2), "tags": []any{"hot", "new"}},
				map[string]any{"id": "b2", "score": float64(5), "tags": []any{"hot"}},
			},
		},
	}
	trueValue := true
	minItems := 1
	minScore := float64(1)
	maxScore := float64(5)
	assertions := []JSONAssertion{
		{Path: "data.items[*].id", Matches: "^[a-z][0-9]$"},
		{Path: "data.items[*].id", NonEmpty: &trueValue},
		{Path: "data.items[*].score", Min: &minScore, Max: &maxScore},
		{Path: "data.items[*].tags", MinItems: &minItems, Contains: "hot", AllType: "string", AllNonNull: &trueValue},
	}
	for _, assertion := range assertions {
		if failure := evaluateAssertion(decoded, assertion); failure != "" {
			t.Fatalf("evaluateAssertion(%#v) failure = %s", assertion, failure)
		}
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

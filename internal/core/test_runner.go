package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/MarlonJD/graphgate/internal/config"
)

type TestOptions struct {
	Environment string
	Update      bool
	Timeout     time.Duration
}

type Fixture struct {
	Name                 string            `json:"name,omitempty"`
	Operation            string            `json:"operation,omitempty"`
	OperationName        string            `json:"operationName,omitempty"`
	RequestOperationName string            `json:"requestOperationName,omitempty"`
	IncludeQuery         *bool             `json:"includeQuery,omitempty"`
	Headers              map[string]string `json:"headers,omitempty"`
	Variables            map[string]any    `json:"variables,omitempty"`
	ExpectedStatus       int               `json:"expectedStatus,omitempty"`
	ExpectedErrorCodes   []string          `json:"expectedErrorCodes,omitempty"`
	Assertions           []JSONAssertion   `json:"assertions,omitempty"`
	Snapshot             any               `json:"snapshot,omitempty"`
}

type JSONAssertion struct {
	Path     string `json:"path"`
	Exists   *bool  `json:"exists,omitempty"`
	Type     string `json:"type,omitempty"`
	Equals   any    `json:"equals,omitempty"`
	MinItems *int   `json:"minItems,omitempty"`
}

type TestRunResult struct {
	OK          bool               `json:"ok"`
	Environment string             `json:"environment"`
	Endpoint    string             `json:"endpoint"`
	Passed      int                `json:"passed"`
	Failed      int                `json:"failed"`
	Results     []FixtureRunResult `json:"results"`
}

type FixtureRunResult struct {
	Name                 string   `json:"name"`
	File                 string   `json:"file"`
	Operation            string   `json:"operation"`
	RequestOperationName string   `json:"requestOperationName,omitempty"`
	Status               int      `json:"status"`
	Passed               bool     `json:"passed"`
	Failures             []string `json:"failures"`
	ErrorCodes           []string `json:"errorCodes,omitempty"`
	SnapshotUpdate       bool     `json:"snapshotUpdate,omitempty"`
}

func RunContractTests(ctx context.Context, cfg *config.Config, options TestOptions) (TestRunResult, error) {
	if options.Environment == "" {
		options.Environment = "local"
	}
	if options.Timeout <= 0 {
		options.Timeout = 10 * time.Second
	}
	env, ok := cfg.Environments[options.Environment]
	if !ok {
		return TestRunResult{}, fmt.Errorf("environment %q not found", options.Environment)
	}
	if strings.TrimSpace(env.Endpoint) == "" {
		return TestRunResult{}, fmt.Errorf("environment %q endpoint is required", options.Environment)
	}

	validation, err := ValidateProject(cfg)
	if err != nil {
		return TestRunResult{}, err
	}
	if !validation.OK() {
		return TestRunResult{}, fmt.Errorf("project validation failed before test run")
	}
	operations := map[string]Operation{}
	for _, op := range validation.Operations {
		operations[op.Name] = op
	}

	files, err := cfg.FixtureFiles()
	if err != nil {
		return TestRunResult{}, err
	}
	if cfg.Tests.RequireOperationCoverage {
		if err := ensureFixtureOperationCoverage(cfg, files, operations); err != nil {
			return TestRunResult{}, err
		}
	}

	result := TestRunResult{
		OK:          true,
		Environment: options.Environment,
		Endpoint:    env.Endpoint,
		Results:     []FixtureRunResult{},
	}
	client := &http.Client{Timeout: options.Timeout}
	for _, file := range files {
		run, err := runFixture(ctx, cfg, client, env, operations, file, options.Update)
		if err != nil {
			return result, err
		}
		if run.Passed {
			result.Passed++
		} else {
			result.Failed++
			result.OK = false
		}
		result.Results = append(result.Results, run)
	}
	return result, nil
}

func runFixture(ctx context.Context, cfg *config.Config, client *http.Client, env config.EnvironmentConfig, operations map[string]Operation, file string, update bool) (FixtureRunResult, error) {
	fixture, err := readFixture(cfg, file)
	if err != nil {
		return FixtureRunResult{}, err
	}

	operationName := fixtureOperationName(fixture)
	run := FixtureRunResult{
		Name:                 fixtureName(file, fixture),
		File:                 cfg.RelativePath(file),
		Operation:            operationName,
		RequestOperationName: fixture.RequestOperationName,
	}
	operation, ok := operations[operationName]
	if !ok {
		run.Failures = append(run.Failures, fmt.Sprintf("operation %q not found", operationName))
		return finishFixtureRun(run), nil
	}
	run.RequestOperationName = requestOperationName(fixture, operation)

	payload := map[string]any{
		"operationName": requestOperationName(fixture, operation),
		"variables":     expandValue(fixture.Variables),
	}
	if includeQuery(fixture) {
		payload["query"] = operation.Normalized
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return run, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, env.Endpoint, bytes.NewReader(body))
	if err != nil {
		return run, err
	}
	req.Header.Set("Content-Type", "application/json")
	for name, value := range env.Headers {
		req.Header.Set(name, os.ExpandEnv(value))
	}
	for name, value := range fixture.Headers {
		req.Header.Set(name, os.ExpandEnv(value))
	}

	resp, err := client.Do(req)
	if err != nil {
		run.Failures = append(run.Failures, err.Error())
		return finishFixtureRun(run), nil
	}
	defer resp.Body.Close()
	run.Status = resp.StatusCode
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return run, err
	}
	var decoded any
	if len(bytes.TrimSpace(respBody)) > 0 {
		if err := json.Unmarshal(respBody, &decoded); err != nil {
			run.Failures = append(run.Failures, "response is not valid JSON")
			return finishFixtureRun(run), nil
		}
	}

	expectedStatus := fixture.ExpectedStatus
	if expectedStatus == 0 {
		expectedStatus = http.StatusOK
	}
	if run.Status != expectedStatus {
		run.Failures = append(run.Failures, fmt.Sprintf("status = %d, want %d", run.Status, expectedStatus))
	}

	run.ErrorCodes = graphqlErrorCodes(decoded)
	if !sameStringSet(run.ErrorCodes, fixture.ExpectedErrorCodes) {
		run.Failures = append(run.Failures, fmt.Sprintf("GraphQL error codes = %v, want %v", run.ErrorCodes, fixture.ExpectedErrorCodes))
	}
	for _, assertion := range fixture.Assertions {
		if failure := evaluateAssertion(decoded, assertion); failure != "" {
			run.Failures = append(run.Failures, failure)
		}
	}
	if fixture.Snapshot != nil && !reflect.DeepEqual(fixture.Snapshot, decoded) {
		run.Failures = append(run.Failures, "response differs from snapshot")
	}
	if update {
		fixture.Snapshot = decoded
		updated, err := json.MarshalIndent(fixture, "", "  ")
		if err != nil {
			return run, err
		}
		if err := os.WriteFile(file, append(updated, '\n'), 0o644); err != nil {
			return run, err
		}
		run.SnapshotUpdate = true
	}
	return finishFixtureRun(run), nil
}

func readFixture(cfg *config.Config, file string) (Fixture, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return Fixture{}, err
	}
	var fixture Fixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return Fixture{}, fmt.Errorf("parse fixture %s: %w", cfg.RelativePath(file), err)
	}
	return fixture, nil
}

func finishFixtureRun(run FixtureRunResult) FixtureRunResult {
	run.Passed = len(run.Failures) == 0
	return run
}

func ensureFixtureOperationCoverage(cfg *config.Config, files []string, operations map[string]Operation) error {
	covered := map[string]struct{}{}
	for _, file := range files {
		fixture, err := readFixture(cfg, file)
		if err != nil {
			return err
		}
		operationName := fixtureOperationName(fixture)
		if operationName != "" {
			covered[operationName] = struct{}{}
		}
	}

	var missing []string
	for name := range operations {
		if _, ok := covered[name]; !ok {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		return fmt.Errorf("fixtures do not cover operations: %s", strings.Join(missing, ", "))
	}
	return nil
}

func fixtureOperationName(fixture Fixture) string {
	if fixture.OperationName != "" {
		return fixture.OperationName
	}
	return fixture.Operation
}

func fixtureName(file string, fixture Fixture) string {
	if fixture.Name != "" {
		return fixture.Name
	}
	base := filepath.Base(file)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func requestOperationName(fixture Fixture, operation Operation) string {
	if fixture.RequestOperationName != "" {
		return fixture.RequestOperationName
	}
	return operation.Name
}

func includeQuery(fixture Fixture) bool {
	return fixture.IncludeQuery == nil || *fixture.IncludeQuery
}

func graphqlErrorCodes(decoded any) []string {
	obj, ok := decoded.(map[string]any)
	if !ok {
		return nil
	}
	rawErrors, ok := obj["errors"].([]any)
	if !ok {
		return nil
	}
	var codes []string
	for _, rawErr := range rawErrors {
		errObj, ok := rawErr.(map[string]any)
		if !ok {
			continue
		}
		extensions, ok := errObj["extensions"].(map[string]any)
		if !ok {
			continue
		}
		if code, ok := extensions["code"].(string); ok {
			codes = append(codes, code)
		}
	}
	return codes
}

func sameStringSet(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	counts := map[string]int{}
	for _, value := range left {
		counts[value]++
	}
	for _, value := range right {
		counts[value]--
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}

func evaluateAssertion(decoded any, assertion JSONAssertion) string {
	value, exists := valueAtPath(decoded, assertion.Path)
	if assertion.Exists != nil && *assertion.Exists != exists {
		return fmt.Sprintf("%s exists = %v, want %v", assertion.Path, exists, *assertion.Exists)
	}
	if assertion.Type != "" && exists && valueType(value) != assertion.Type {
		return fmt.Sprintf("%s type = %s, want %s", assertion.Path, valueType(value), assertion.Type)
	}
	if assertion.Equals != nil && !reflect.DeepEqual(value, assertion.Equals) {
		return fmt.Sprintf("%s = %v, want %v", assertion.Path, value, assertion.Equals)
	}
	if assertion.MinItems != nil {
		items, ok := value.([]any)
		if !ok {
			return fmt.Sprintf("%s type = %s, want array for minItems", assertion.Path, valueType(value))
		}
		if len(items) < *assertion.MinItems {
			return fmt.Sprintf("%s items = %d, want at least %d", assertion.Path, len(items), *assertion.MinItems)
		}
	}
	return ""
}

func expandValue(value any) any {
	switch typed := value.(type) {
	case string:
		return os.ExpandEnv(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = expandValue(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = expandValue(item)
		}
		return out
	default:
		return value
	}
}

func valueAtPath(decoded any, path string) (any, bool) {
	if path == "" {
		return decoded, true
	}
	current := decoded
	for _, part := range strings.Split(path, ".") {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = obj[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func valueType(value any) string {
	switch value.(type) {
	case nil:
		return "null"
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return fmt.Sprintf("%T", value)
	}
}

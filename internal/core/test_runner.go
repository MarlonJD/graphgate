package core

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/MarlonJD/graphgate/internal/config"
)

var (
	ErrUnknownSuite       = errors.New("unknown fixture suite")
	ErrNoSelectedFixtures = errors.New("selected fixture set is empty")
)

const (
	FailureClassReadiness        = "readiness_failed"
	FailureClassCoverage         = "coverage_failed"
	FailureClassUnknownOperation = "unknown_operation"
	FailureClassStatusMismatch   = "status_mismatch"
	FailureClassGraphQLError     = "graphql_error"
	FailureClassAssertion        = "assertion_failed"
	FailureClassSnapshot         = "snapshot_mismatch"
	FailureClassLatency          = "latency_exceeded"
	FailureClassInvalidJSON      = "invalid_json"
	FailureClassNetwork          = "network_error"
	FailureClassTimeout          = "timeout"
)

type TestOptions struct {
	Environment      string
	Update           bool
	Timeout          time.Duration
	Tags             []string
	Exclude          []string
	Suite            string
	SkipReadiness    bool
	Command          []string
	GraphGateVersion string
}

type Fixture struct {
	Name                 string            `json:"name,omitempty"`
	Operation            string            `json:"operation,omitempty"`
	OperationName        string            `json:"operationName,omitempty"`
	RequestOperationName string            `json:"requestOperationName,omitempty"`
	IncludeQuery         *bool             `json:"includeQuery,omitempty"`
	Tags                 []string          `json:"tags,omitempty"`
	Profile              string            `json:"profile,omitempty"`
	Profiles             []string          `json:"profiles,omitempty"`
	Headers              map[string]string `json:"headers,omitempty"`
	Variables            map[string]any    `json:"variables,omitempty"`
	ExpectedStatus       int               `json:"expectedStatus,omitempty"`
	ExpectedErrorCodes   []string          `json:"expectedErrorCodes,omitempty"`
	ExpectedFailureClass string            `json:"expectedFailureClass,omitempty"`
	Assertions           []JSONAssertion   `json:"assertions,omitempty"`
	Snapshot             any               `json:"snapshot,omitempty"`
	SnapshotMode         string            `json:"snapshotMode,omitempty"`
	SnapshotIgnorePaths  []string          `json:"snapshotIgnorePaths,omitempty"`
	SnapshotRedactPaths  []string          `json:"snapshotRedactPaths,omitempty"`
	MaxLatencyMs         int               `json:"maxLatencyMs,omitempty"`
}

type JSONAssertion struct {
	Path       string   `json:"path"`
	Exists     *bool    `json:"exists,omitempty"`
	Type       string   `json:"type,omitempty"`
	Equals     any      `json:"equals,omitempty"`
	MinItems   *int     `json:"minItems,omitempty"`
	NonEmpty   *bool    `json:"nonEmpty,omitempty"`
	Matches    string   `json:"matches,omitempty"`
	Contains   any      `json:"contains,omitempty"`
	AllType    string   `json:"allType,omitempty"`
	AllNonNull *bool    `json:"allNonNull,omitempty"`
	Min        *float64 `json:"min,omitempty"`
	Max        *float64 `json:"max,omitempty"`
}

type TestRunResult struct {
	OK           bool               `json:"ok"`
	Environment  string             `json:"environment"`
	Endpoint     string             `json:"endpoint"`
	FailureClass string             `json:"failureClass,omitempty"`
	Failures     []string           `json:"failures,omitempty"`
	Metadata     RunMetadata        `json:"metadata"`
	Selection    FixtureSelection   `json:"selection"`
	Readiness    []ReadinessResult  `json:"readiness,omitempty"`
	Timing       TimingSummary      `json:"timing,omitempty"`
	Passed       int                `json:"passed"`
	Failed       int                `json:"failed"`
	Results      []FixtureRunResult `json:"results"`
}

type RunMetadata struct {
	StartedAt        string   `json:"startedAt"`
	CompletedAt      string   `json:"completedAt"`
	DurationMs       int64    `json:"durationMs"`
	Command          []string `json:"command,omitempty"`
	GraphGateVersion string   `json:"graphgateVersion,omitempty"`
	ConfigPath       string   `json:"configPath"`
	ConfigSHA256     string   `json:"configSha256,omitempty"`
	SchemaPath       string   `json:"schemaPath"`
	SchemaSHA256     string   `json:"schemaSha256,omitempty"`
	ManifestPath     string   `json:"manifestPath"`
	ManifestSHA256   string   `json:"manifestSha256,omitempty"`
}

type FixtureSelection struct {
	Suite        string   `json:"suite,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Exclude      []string `json:"exclude,omitempty"`
	FixtureCount int      `json:"fixtureCount"`
}

type ReadinessResult struct {
	Name         string `json:"name"`
	Passed       bool   `json:"passed"`
	FailureClass string `json:"failureClass,omitempty"`
	Message      string `json:"message,omitempty"`
	DurationMs   int64  `json:"durationMs,omitempty"`
}

type TimingSummary struct {
	MinMs int64 `json:"minMs,omitempty"`
	MaxMs int64 `json:"maxMs,omitempty"`
	P50Ms int64 `json:"p50Ms,omitempty"`
	P95Ms int64 `json:"p95Ms,omitempty"`
}

type FixtureRunResult struct {
	Name                 string            `json:"name"`
	File                 string            `json:"file"`
	Operation            string            `json:"operation"`
	RequestOperationName string            `json:"requestOperationName,omitempty"`
	Tags                 []string          `json:"tags,omitempty"`
	Profiles             []string          `json:"profiles,omitempty"`
	Status               int               `json:"status"`
	Passed               bool              `json:"passed"`
	FailureClass         string            `json:"failureClass,omitempty"`
	Failures             []string          `json:"failures"`
	ErrorCodes           []string          `json:"errorCodes,omitempty"`
	DurationMs           int64             `json:"durationMs,omitempty"`
	Attempts             int               `json:"attempts,omitempty"`
	AttemptResults       []FixtureAttempt  `json:"attemptResults,omitempty"`
	Assertions           []AssertionResult `json:"assertions,omitempty"`
	SnapshotUpdate       bool              `json:"snapshotUpdate,omitempty"`
}

type FixtureAttempt struct {
	Attempt      int      `json:"attempt"`
	Status       int      `json:"status,omitempty"`
	FailureClass string   `json:"failureClass,omitempty"`
	Failures     []string `json:"failures,omitempty"`
	DurationMs   int64    `json:"durationMs,omitempty"`
}

type AssertionResult struct {
	Path    string `json:"path"`
	Passed  bool   `json:"passed"`
	Failure string `json:"failure,omitempty"`
}

func RunContractTests(ctx context.Context, cfg *config.Config, options TestOptions) (TestRunResult, error) {
	started := time.Now().UTC()
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

	result := TestRunResult{
		OK:          true,
		Environment: options.Environment,
		Endpoint:    os.ExpandEnv(env.Endpoint),
		Results:     []FixtureRunResult{},
		Metadata:    runMetadata(cfg, options, started),
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
	selectedFiles, selection, err := selectFixtureFiles(cfg, files, options)
	if err != nil {
		return TestRunResult{}, err
	}
	result.Selection = selection

	if failures := ensureFixtureCoverage(cfg, selectedFiles, operations, selection); len(failures) > 0 {
		result.OK = false
		result.FailureClass = FailureClassCoverage
		result.Failures = failures
		return finishRun(result), nil
	}

	if !options.SkipReadiness {
		result.Readiness = runReadiness(ctx, env, result.Endpoint, options.Timeout)
		if readinessFailed(result.Readiness) {
			result.OK = false
			result.FailureClass = FailureClassReadiness
			return finishRun(result), nil
		}
	}

	client := &http.Client{Timeout: options.Timeout}
	for _, file := range selectedFiles {
		run, err := runFixtureWithRetry(ctx, cfg, client, env, operations, file, options, suiteMaxLatency(cfg, options.Suite))
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
	return finishRun(result), nil
}

func runMetadata(cfg *config.Config, options TestOptions, started time.Time) RunMetadata {
	schemaPath := cfg.ResolvePath(cfg.Schema)
	manifestPath := cfg.ManifestOutputPath()
	return RunMetadata{
		StartedAt:        started.Format(time.RFC3339Nano),
		Command:          append([]string(nil), options.Command...),
		GraphGateVersion: options.GraphGateVersion,
		ConfigPath:       cfg.Path,
		ConfigSHA256:     fileSHA256(cfg.Path),
		SchemaPath:       schemaPath,
		SchemaSHA256:     fileSHA256(schemaPath),
		ManifestPath:     manifestPath,
		ManifestSHA256:   fileSHA256(manifestPath),
	}
}

func fileSHA256(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func selectFixtureFiles(cfg *config.Config, files []string, options TestOptions) ([]string, FixtureSelection, error) {
	includeTags := append([]string(nil), options.Tags...)
	excludeTags := append([]string(nil), options.Exclude...)
	if options.Suite != "" {
		suite, ok := cfg.Tests.Suites[options.Suite]
		if !ok {
			return nil, FixtureSelection{}, fmt.Errorf("%w: %s", ErrUnknownSuite, options.Suite)
		}
		includeTags = append(includeTags, suite.Tags...)
		excludeTags = append(excludeTags, suite.Exclude...)
	}
	includeTags = uniqueStrings(includeTags)
	excludeTags = uniqueStrings(excludeTags)

	var selected []string
	for _, file := range files {
		fixture, err := readFixture(cfg, file)
		if err != nil {
			return nil, FixtureSelection{}, err
		}
		if fixtureMatchesTags(fixture.Tags, includeTags, excludeTags) {
			selected = append(selected, file)
		}
	}
	sort.Strings(selected)
	selection := FixtureSelection{
		Suite:        options.Suite,
		Tags:         includeTags,
		Exclude:      excludeTags,
		FixtureCount: len(selected),
	}
	if len(selected) == 0 {
		return nil, selection, fmt.Errorf("%w: suite=%q tags=%v exclude=%v", ErrNoSelectedFixtures, options.Suite, includeTags, excludeTags)
	}
	return selected, selection, nil
}

func fixtureMatchesTags(fixtureTags []string, includeTags []string, excludeTags []string) bool {
	tagSet := stringSet(fixtureTags)
	for _, tag := range includeTags {
		if _, ok := tagSet[tag]; !ok {
			return false
		}
	}
	for _, tag := range excludeTags {
		if _, ok := tagSet[tag]; ok {
			return false
		}
	}
	return true
}

func runReadiness(ctx context.Context, env config.EnvironmentConfig, endpoint string, defaultTimeout time.Duration) []ReadinessResult {
	var results []ReadinessResult
	for _, name := range env.RequiredEnv {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		value, ok := os.LookupEnv(name)
		result := ReadinessResult{Name: "env:" + name, Passed: ok && value != ""}
		if !result.Passed {
			result.FailureClass = FailureClassReadiness
			result.Message = fmt.Sprintf("required environment variable %s is not set", name)
		}
		results = append(results, result)
	}
	if env.HealthCheck.URL != "" || env.HealthCheck.Path != "" {
		results = append(results, runHealthCheck(ctx, env, endpoint, defaultTimeout))
	}
	return results
}

func runHealthCheck(ctx context.Context, env config.EnvironmentConfig, endpoint string, defaultTimeout time.Duration) ReadinessResult {
	started := time.Now()
	result := ReadinessResult{Name: "healthCheck"}
	timeout := defaultTimeout
	if env.HealthCheck.Timeout != "" {
		if parsed, err := time.ParseDuration(os.ExpandEnv(env.HealthCheck.Timeout)); err == nil && parsed > 0 {
			timeout = parsed
		}
	}
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	method := strings.TrimSpace(env.HealthCheck.Method)
	if method == "" {
		method = http.MethodGet
	}
	checkURL := os.ExpandEnv(env.HealthCheck.URL)
	if checkURL == "" {
		checkURL = healthCheckURL(endpoint, os.ExpandEnv(env.HealthCheck.Path))
	}
	expectedStatus := env.HealthCheck.ExpectedStatus
	if expectedStatus == 0 {
		expectedStatus = http.StatusOK
	}
	req, err := http.NewRequestWithContext(checkCtx, method, checkURL, nil)
	if err != nil {
		result.FailureClass = FailureClassReadiness
		result.Message = err.Error()
		result.DurationMs = time.Since(started).Milliseconds()
		return result
	}
	for name, value := range sortedHeaderPairs(env.Headers) {
		req.Header.Set(name, os.ExpandEnv(value))
	}
	for name, value := range sortedHeaderPairs(env.HealthCheck.Headers) {
		req.Header.Set(name, os.ExpandEnv(value))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		result.FailureClass = classifyRequestError(err)
		if result.FailureClass == "" {
			result.FailureClass = FailureClassReadiness
		}
		result.Message = err.Error()
		result.DurationMs = time.Since(started).Milliseconds()
		return result
	}
	defer resp.Body.Close()
	result.DurationMs = time.Since(started).Milliseconds()
	result.Passed = resp.StatusCode == expectedStatus
	if !result.Passed {
		result.FailureClass = FailureClassReadiness
		result.Message = fmt.Sprintf("status = %d, want %d", resp.StatusCode, expectedStatus)
	}
	return result
}

func healthCheckURL(endpoint string, path string) string {
	if strings.TrimSpace(path) == "" {
		return endpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}
	parsed.Path = path
	parsed.RawQuery = ""
	return parsed.String()
}

func readinessFailed(results []ReadinessResult) bool {
	for _, result := range results {
		if !result.Passed {
			return true
		}
	}
	return false
}

func runFixtureWithRetry(ctx context.Context, cfg *config.Config, client *http.Client, env config.EnvironmentConfig, operations map[string]Operation, file string, options TestOptions, suiteLatencyMs int) (FixtureRunResult, error) {
	retry := retryPolicy(cfg.Tests.Retry)
	var last FixtureRunResult
	for attempt := 1; attempt <= retry.maxAttempts; attempt++ {
		run, err := runFixtureAttempt(ctx, cfg, client, env, operations, file, options.Update, suiteLatencyMs)
		if err != nil {
			return run, err
		}
		run.Attempts = attempt
		run.AttemptResults = append(last.AttemptResults, FixtureAttempt{
			Attempt:      attempt,
			Status:       run.Status,
			FailureClass: run.FailureClass,
			Failures:     append([]string(nil), run.Failures...),
			DurationMs:   run.DurationMs,
		})
		last = run
		if run.Passed || !retry.retryable[run.FailureClass] || attempt == retry.maxAttempts {
			return run, nil
		}
		if retry.backoff > 0 {
			timer := time.NewTimer(retry.backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return run, ctx.Err()
			case <-timer.C:
			}
		}
	}
	return last, nil
}

func runFixtureAttempt(ctx context.Context, cfg *config.Config, client *http.Client, env config.EnvironmentConfig, operations map[string]Operation, file string, update bool, suiteLatencyMs int) (FixtureRunResult, error) {
	fixture, err := readFixture(cfg, file)
	if err != nil {
		return FixtureRunResult{}, err
	}

	operationName := fixtureOperationName(fixture)
	profiles := fixtureProfileNames(fixture)
	run := FixtureRunResult{
		Name:                 fixtureName(file, fixture),
		File:                 cfg.RelativePath(file),
		Operation:            operationName,
		RequestOperationName: fixture.RequestOperationName,
		Tags:                 append([]string(nil), fixture.Tags...),
		Profiles:             profiles,
		Failures:             []string{},
	}
	operation, ok := operations[operationName]
	if !ok {
		run.Failures = append(run.Failures, fmt.Sprintf("operation %q not found", operationName))
		run.FailureClass = FailureClassUnknownOperation
		return finishFixtureRun(run, fixture), nil
	}
	run.RequestOperationName = requestOperationName(fixture, operation)

	variables, err := mergedVariables(cfg, profiles, fixture)
	if err != nil {
		run.Failures = append(run.Failures, err.Error())
		run.FailureClass = FailureClassUnknownOperation
		return finishFixtureRun(run, fixture), nil
	}
	payload := map[string]any{
		"operationName": run.RequestOperationName,
		"variables":     expandValue(variables),
	}
	if includeQuery(fixture) {
		payload["query"] = operation.Normalized
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return run, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, os.ExpandEnv(env.Endpoint), bytes.NewReader(body))
	if err != nil {
		return run, err
	}
	req.Header.Set("Content-Type", "application/json")
	headers, err := mergedHeaders(cfg, env, profiles, fixture)
	if err != nil {
		run.Failures = append(run.Failures, err.Error())
		run.FailureClass = FailureClassUnknownOperation
		return finishFixtureRun(run, fixture), nil
	}
	for name, value := range sortedHeaderPairs(headers) {
		req.Header.Set(name, os.ExpandEnv(value))
	}

	started := time.Now()
	resp, err := client.Do(req)
	run.DurationMs = time.Since(started).Milliseconds()
	if err != nil {
		run.Failures = append(run.Failures, err.Error())
		run.FailureClass = classifyRequestError(err)
		return finishFixtureRun(run, fixture), nil
	}
	defer resp.Body.Close()
	run.Status = resp.StatusCode
	respBody, err := io.ReadAll(resp.Body)
	run.DurationMs = time.Since(started).Milliseconds()
	if err != nil {
		return run, err
	}
	var decoded any
	if len(bytes.TrimSpace(respBody)) > 0 {
		if err := json.Unmarshal(respBody, &decoded); err != nil {
			run.Failures = append(run.Failures, "response is not valid JSON")
			run.FailureClass = FailureClassInvalidJSON
			return finishFixtureRun(run, fixture), nil
		}
	}

	expectedStatus := fixture.ExpectedStatus
	if expectedStatus == 0 {
		expectedStatus = http.StatusOK
	}
	if run.Status != expectedStatus {
		run.Failures = append(run.Failures, fmt.Sprintf("status = %d, want %d", run.Status, expectedStatus))
		setFailureClass(&run, FailureClassStatusMismatch)
	}

	run.ErrorCodes = graphqlErrorCodes(decoded)
	if !sameStringSet(run.ErrorCodes, fixture.ExpectedErrorCodes) {
		run.Failures = append(run.Failures, fmt.Sprintf("GraphQL error codes = %v, want %v", run.ErrorCodes, fixture.ExpectedErrorCodes))
		setFailureClass(&run, FailureClassGraphQLError)
	}
	for _, assertion := range fixture.Assertions {
		assertionResult := AssertionResult{Path: assertion.Path, Passed: true}
		if failure := evaluateAssertion(decoded, assertion); failure != "" {
			assertionResult.Passed = false
			assertionResult.Failure = failure
			run.Failures = append(run.Failures, failure)
			setFailureClass(&run, FailureClassAssertion)
		}
		run.Assertions = append(run.Assertions, assertionResult)
	}
	if failure, snapshotUpdate, err := evaluateSnapshot(cfg, file, fixture, decoded, update); err != nil {
		return run, err
	} else if failure != "" {
		run.Failures = append(run.Failures, failure)
		setFailureClass(&run, FailureClassSnapshot)
	} else if snapshotUpdate {
		run.SnapshotUpdate = true
	}
	if maxLatency := fixtureMaxLatency(cfg, fixture, suiteLatencyMs); maxLatency > 0 && run.DurationMs > int64(maxLatency) {
		run.Failures = append(run.Failures, fmt.Sprintf("duration = %dms, want <= %dms", run.DurationMs, maxLatency))
		setFailureClass(&run, FailureClassLatency)
	}
	return finishFixtureRun(run, fixture), nil
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

func finishFixtureRun(run FixtureRunResult, fixture Fixture) FixtureRunResult {
	run.Passed = len(run.Failures) == 0
	if run.Passed && fixture.ExpectedFailureClass != "" {
		run.FailureClass = fixture.ExpectedFailureClass
	}
	return run
}

func finishRun(result TestRunResult) TestRunResult {
	completed := time.Now().UTC()
	if result.Metadata.CompletedAt == "" {
		result.Metadata.CompletedAt = completed.Format(time.RFC3339Nano)
	}
	if result.Metadata.DurationMs == 0 && result.Metadata.StartedAt != "" {
		if started, err := time.Parse(time.RFC3339Nano, result.Metadata.StartedAt); err == nil {
			result.Metadata.DurationMs = completed.Sub(started).Milliseconds()
		}
	}
	if len(result.Results) > 0 {
		result.Timing = timingSummary(result.Results)
	}
	return result
}

func ensureFixtureCoverage(cfg *config.Config, files []string, operations map[string]Operation, selection FixtureSelection) []string {
	coverageOperations := operations
	if selection.Suite != "" || len(selection.Tags) > 0 || len(selection.Exclude) > 0 {
		coverageOperations = selectedOperationUniverse(cfg, files, operations)
	}
	var failures []string
	if cfg.Tests.RequireOperationCoverage {
		failures = append(failures, missingOperationCoverage(cfg, files, coverageOperations, false)...)
	}
	if cfg.Tests.Coverage.RequirePositiveFixture && !negativeSelection(selection) {
		failures = append(failures, missingOperationCoverage(cfg, files, coverageOperations, true)...)
	}
	if len(cfg.Tests.Coverage.RequireTags) > 0 {
		failures = append(failures, fixtureRequiredTagFailures(cfg, files, cfg.Tests.Coverage.RequireTags)...)
	}
	if cfg.Tests.Coverage.ForbidUnknownOperations {
		failures = append(failures, unknownOperationFailures(cfg, files, operations)...)
	}
	if cfg.Tests.Coverage.ForbidDeprecatedOperations {
		failures = append(failures, deprecatedOperationFailures(cfg, files, operations)...)
	}
	sort.Strings(failures)
	return failures
}

func negativeSelection(selection FixtureSelection) bool {
	tags := stringSet(selection.Tags)
	_, hasNegative := tags["negative"]
	_, hasPositive := tags["positive"]
	return hasNegative && !hasPositive
}

func selectedOperationUniverse(cfg *config.Config, files []string, operations map[string]Operation) map[string]Operation {
	selected := map[string]Operation{}
	for _, file := range files {
		fixture, err := readFixture(cfg, file)
		if err != nil {
			continue
		}
		operation, ok := operations[fixtureOperationName(fixture)]
		if ok {
			selected[operation.Name] = operation
		}
	}
	return selected
}

func missingOperationCoverage(cfg *config.Config, files []string, operations map[string]Operation, positiveOnly bool) []string {
	covered := map[string]struct{}{}
	for _, file := range files {
		fixture, err := readFixture(cfg, file)
		if err != nil {
			return []string{err.Error()}
		}
		if positiveOnly && !positiveFixture(fixture) {
			continue
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
	if len(missing) == 0 {
		return nil
	}
	if positiveOnly {
		return []string{fmt.Sprintf("positive fixtures do not cover operations: %s", strings.Join(missing, ", "))}
	}
	return []string{fmt.Sprintf("fixtures do not cover operations: %s", strings.Join(missing, ", "))}
}

func fixtureRequiredTagFailures(cfg *config.Config, files []string, tags []string) []string {
	var failures []string
	for _, file := range files {
		fixture, err := readFixture(cfg, file)
		if err != nil {
			return []string{err.Error()}
		}
		tagSet := stringSet(fixture.Tags)
		for _, tag := range tags {
			if _, ok := tagSet[tag]; !ok {
				failures = append(failures, fmt.Sprintf("%s missing required tag %q", cfg.RelativePath(file), tag))
			}
		}
	}
	return failures
}

func unknownOperationFailures(cfg *config.Config, files []string, operations map[string]Operation) []string {
	var failures []string
	for _, file := range files {
		fixture, err := readFixture(cfg, file)
		if err != nil {
			return []string{err.Error()}
		}
		operationName := fixtureOperationName(fixture)
		if operationName == "" {
			failures = append(failures, fmt.Sprintf("%s does not declare an operation", cfg.RelativePath(file)))
			continue
		}
		if _, ok := operations[operationName]; !ok {
			failures = append(failures, fmt.Sprintf("%s references unknown operation %q", cfg.RelativePath(file), operationName))
		}
	}
	return failures
}

func deprecatedOperationFailures(cfg *config.Config, files []string, operations map[string]Operation) []string {
	var failures []string
	for _, file := range files {
		fixture, err := readFixture(cfg, file)
		if err != nil {
			return []string{err.Error()}
		}
		operation, ok := operations[fixtureOperationName(fixture)]
		if ok && len(operation.DeprecatedFields) > 0 {
			failures = append(failures, fmt.Sprintf("%s operation %q uses deprecated fields: %s", cfg.RelativePath(file), operation.Name, strings.Join(operation.DeprecatedFields, ", ")))
		}
	}
	return failures
}

func positiveFixture(fixture Fixture) bool {
	tags := stringSet(fixture.Tags)
	if _, ok := tags["negative"]; ok {
		return false
	}
	expectedStatus := fixture.ExpectedStatus
	if expectedStatus == 0 {
		expectedStatus = http.StatusOK
	}
	return expectedStatus < 400 && len(fixture.ExpectedErrorCodes) == 0
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

func fixtureProfileNames(fixture Fixture) []string {
	var profiles []string
	if strings.TrimSpace(fixture.Profile) != "" {
		profiles = append(profiles, strings.TrimSpace(fixture.Profile))
	}
	profiles = append(profiles, fixture.Profiles...)
	return uniqueStrings(profiles)
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

func mergedHeaders(cfg *config.Config, env config.EnvironmentConfig, profiles []string, fixture Fixture) (map[string]string, error) {
	headers := map[string]string{}
	mergeStringMap(headers, env.Headers)
	for _, name := range profiles {
		profile, ok := cfg.Profiles[name]
		if !ok {
			return nil, fmt.Errorf("profile %q not found", name)
		}
		mergeStringMap(headers, profile.Headers)
	}
	mergeStringMap(headers, fixture.Headers)
	return headers, nil
}

func mergedVariables(cfg *config.Config, profiles []string, fixture Fixture) (map[string]any, error) {
	variables := map[string]any{}
	for _, name := range profiles {
		profile, ok := cfg.Profiles[name]
		if !ok {
			return nil, fmt.Errorf("profile %q not found", name)
		}
		mergeAnyMap(variables, profile.Variables)
	}
	mergeAnyMap(variables, fixture.Variables)
	return variables, nil
}

func mergeStringMap(dst map[string]string, src map[string]string) {
	for key, value := range src {
		dst[key] = value
	}
}

func mergeAnyMap(dst map[string]any, src map[string]any) {
	for key, value := range src {
		dst[key] = value
	}
}

func sortedHeaderPairs(headers map[string]string) map[string]string {
	if len(headers) <= 1 {
		return headers
	}
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make(map[string]string, len(headers))
	for _, key := range keys {
		out[key] = headers[key]
	}
	return out
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
	sort.Strings(codes)
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
	values := valuesAtPath(decoded, assertion.Path)
	exists := len(values) > 0
	if assertion.Exists != nil && *assertion.Exists != exists {
		return fmt.Sprintf("%s exists = %v, want %v", assertion.Path, exists, *assertion.Exists)
	}
	if !exists {
		if assertion.Equals != nil || assertion.MinItems != nil || assertion.NonEmpty != nil || assertion.Matches != "" || assertion.Contains != nil || assertion.AllType != "" || assertion.AllNonNull != nil || assertion.Min != nil || assertion.Max != nil {
			return fmt.Sprintf("%s does not exist", assertion.Path)
		}
		return ""
	}
	for _, item := range values {
		value := item.Value
		if assertion.Type != "" && valueType(value) != assertion.Type {
			return fmt.Sprintf("%s type = %s, want %s", item.Path, valueType(value), assertion.Type)
		}
		if assertion.Equals != nil && !reflect.DeepEqual(value, assertion.Equals) {
			return fmt.Sprintf("%s = %v, want %v", item.Path, value, assertion.Equals)
		}
		if assertion.MinItems != nil {
			items, ok := value.([]any)
			if !ok {
				return fmt.Sprintf("%s type = %s, want array for minItems", item.Path, valueType(value))
			}
			if len(items) < *assertion.MinItems {
				return fmt.Sprintf("%s items = %d, want at least %d", item.Path, len(items), *assertion.MinItems)
			}
		}
		if assertion.NonEmpty != nil {
			if got := nonEmpty(value); got != *assertion.NonEmpty {
				return fmt.Sprintf("%s nonEmpty = %v, want %v", item.Path, got, *assertion.NonEmpty)
			}
		}
		if assertion.Matches != "" {
			text, ok := value.(string)
			if !ok {
				return fmt.Sprintf("%s type = %s, want string for matches", item.Path, valueType(value))
			}
			matched, err := regexp.MatchString(assertion.Matches, text)
			if err != nil {
				return fmt.Sprintf("%s matches pattern is invalid: %v", assertion.Path, err)
			}
			if !matched {
				return fmt.Sprintf("%s = %q, want match %q", item.Path, text, assertion.Matches)
			}
		}
		if assertion.Contains != nil && !containsValue(value, assertion.Contains) {
			return fmt.Sprintf("%s does not contain %v", item.Path, assertion.Contains)
		}
		if assertion.AllType != "" {
			if failure := allTypeFailure(item.Path, value, assertion.AllType); failure != "" {
				return failure
			}
		}
		if assertion.AllNonNull != nil {
			if failure := allNonNullFailure(item.Path, value, *assertion.AllNonNull); failure != "" {
				return failure
			}
		}
		if assertion.Min != nil || assertion.Max != nil {
			number, ok := value.(float64)
			if !ok {
				return fmt.Sprintf("%s type = %s, want number", item.Path, valueType(value))
			}
			if assertion.Min != nil && number < *assertion.Min {
				return fmt.Sprintf("%s = %v, want >= %v", item.Path, number, *assertion.Min)
			}
			if assertion.Max != nil && number > *assertion.Max {
				return fmt.Sprintf("%s = %v, want <= %v", item.Path, number, *assertion.Max)
			}
		}
	}
	return ""
}

func nonEmpty(value any) bool {
	switch typed := value.(type) {
	case string:
		return typed != ""
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	default:
		return value != nil
	}
}

func containsValue(value any, expected any) bool {
	switch typed := value.(type) {
	case string:
		expectedString, ok := expected.(string)
		return ok && strings.Contains(typed, expectedString)
	case []any:
		for _, item := range typed {
			if reflect.DeepEqual(item, expected) {
				return true
			}
		}
		return false
	case map[string]any:
		key, ok := expected.(string)
		if ok {
			_, exists := typed[key]
			return exists
		}
		for _, item := range typed {
			if reflect.DeepEqual(item, expected) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func allTypeFailure(path string, value any, expectedType string) string {
	items, ok := value.([]any)
	if !ok {
		if valueType(value) == expectedType {
			return ""
		}
		return fmt.Sprintf("%s type = %s, want %s", path, valueType(value), expectedType)
	}
	for i, item := range items {
		if valueType(item) != expectedType {
			return fmt.Sprintf("%s[%d] type = %s, want %s", path, i, valueType(item), expectedType)
		}
	}
	return ""
}

func allNonNullFailure(path string, value any, expected bool) string {
	items, ok := value.([]any)
	if !ok {
		got := value != nil
		if got != expected {
			return fmt.Sprintf("%s nonNull = %v, want %v", path, got, expected)
		}
		return ""
	}
	for i, item := range items {
		got := item != nil
		if got != expected {
			return fmt.Sprintf("%s[%d] nonNull = %v, want %v", path, i, got, expected)
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
	values := valuesAtPath(decoded, path)
	if len(values) == 0 {
		return nil, false
	}
	return values[0].Value, true
}

type pathValue struct {
	Path  string
	Value any
}

func valuesAtPath(decoded any, path string) []pathValue {
	tokens := parsePath(path)
	return walkPath([]pathValue{{Path: "", Value: decoded}}, tokens)
}

func walkPath(values []pathValue, tokens []pathToken) []pathValue {
	if len(tokens) == 0 {
		return values
	}
	token := tokens[0]
	var next []pathValue
	for _, item := range values {
		switch typed := item.Value.(type) {
		case map[string]any:
			if token.Wildcard {
				keys := make([]string, 0, len(typed))
				for key := range typed {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				for _, key := range keys {
					next = append(next, pathValue{Path: joinPath(item.Path, key), Value: typed[key]})
				}
				continue
			}
			if token.Key != "" {
				if value, ok := typed[token.Key]; ok {
					next = append(next, pathValue{Path: joinPath(item.Path, token.Key), Value: value})
				}
			}
		case []any:
			if token.Wildcard {
				for i, value := range typed {
					next = append(next, pathValue{Path: fmt.Sprintf("%s[%d]", item.Path, i), Value: value})
				}
				continue
			}
			if token.Index != nil && *token.Index >= 0 && *token.Index < len(typed) {
				next = append(next, pathValue{Path: fmt.Sprintf("%s[%d]", item.Path, *token.Index), Value: typed[*token.Index]})
			}
		}
	}
	return walkPath(next, tokens[1:])
}

func joinPath(prefix string, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

type pathToken struct {
	Key      string
	Index    *int
	Wildcard bool
}

func parsePath(path string) []pathToken {
	if path == "" {
		return nil
	}
	var tokens []pathToken
	for _, part := range strings.Split(path, ".") {
		for part != "" {
			bracket := strings.Index(part, "[")
			if bracket < 0 {
				if part == "*" {
					tokens = append(tokens, pathToken{Wildcard: true})
				} else {
					tokens = append(tokens, pathToken{Key: part})
				}
				break
			}
			if bracket > 0 {
				tokens = append(tokens, pathToken{Key: part[:bracket]})
			}
			end := strings.Index(part[bracket:], "]")
			if end < 0 {
				tokens = append(tokens, pathToken{Key: part[bracket:]})
				break
			}
			rawIndex := part[bracket+1 : bracket+end]
			if rawIndex == "*" {
				tokens = append(tokens, pathToken{Wildcard: true})
			} else if index, err := strconv.Atoi(rawIndex); err == nil {
				tokens = append(tokens, pathToken{Index: &index})
			}
			part = part[bracket+end+1:]
		}
	}
	return tokens
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

func evaluateSnapshot(cfg *config.Config, file string, fixture Fixture, decoded any, update bool) (string, bool, error) {
	mode := snapshotMode(cfg, fixture)
	if mode == "disabled" {
		return "", false, nil
	}
	ignorePaths := append([]string{}, cfg.Tests.Snapshot.IgnorePaths...)
	ignorePaths = append(ignorePaths, fixture.SnapshotIgnorePaths...)
	redactPaths := append([]string{}, cfg.Tests.Snapshot.RedactPaths...)
	redactPaths = append(redactPaths, fixture.SnapshotRedactPaths...)
	actual := sanitizeSnapshot(decoded, ignorePaths, redactPaths)
	if mode == "shape" {
		actual = shapeOf(actual)
	}
	if fixture.Snapshot != nil && !reflect.DeepEqual(sanitizeSnapshot(fixture.Snapshot, ignorePaths, redactPaths), actual) {
		if mode == "shape" {
			if reflect.DeepEqual(shapeOf(sanitizeSnapshot(fixture.Snapshot, ignorePaths, redactPaths)), actual) {
				return "", false, nil
			}
		}
		return "response differs from snapshot", false, nil
	}
	if update {
		fixture.Snapshot = actual
		updated, err := json.MarshalIndent(fixture, "", "  ")
		if err != nil {
			return "", false, err
		}
		if err := os.WriteFile(file, append(updated, '\n'), 0o644); err != nil {
			return "", false, err
		}
		return "", true, nil
	}
	return "", false, nil
}

func snapshotMode(cfg *config.Config, fixture Fixture) string {
	mode := strings.ToLower(strings.TrimSpace(fixture.SnapshotMode))
	if mode == "" {
		mode = strings.ToLower(strings.TrimSpace(cfg.Tests.Snapshot.Mode))
	}
	if mode == "" {
		return "exact"
	}
	switch mode {
	case "exact", "shape", "disabled":
		return mode
	default:
		return "exact"
	}
}

func sanitizeSnapshot(value any, ignorePaths []string, redactPaths []string) any {
	copied := deepCopyJSON(value)
	for _, path := range ignorePaths {
		removePath(copied, parsePath(path))
	}
	for _, path := range redactPaths {
		setPath(copied, parsePath(path), "[REDACTED]")
	}
	return copied
}

func deepCopyJSON(value any) any {
	data, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		return value
	}
	return out
}

func removePath(value any, tokens []pathToken) {
	if len(tokens) == 0 {
		return
	}
	token := tokens[0]
	last := len(tokens) == 1
	switch typed := value.(type) {
	case map[string]any:
		if token.Wildcard {
			for _, item := range typed {
				removePath(item, tokens[1:])
			}
			return
		}
		if last {
			delete(typed, token.Key)
			return
		}
		removePath(typed[token.Key], tokens[1:])
	case []any:
		if token.Wildcard {
			for _, item := range typed {
				removePath(item, tokens[1:])
			}
			return
		}
		if token.Index != nil && *token.Index >= 0 && *token.Index < len(typed) {
			if last {
				typed[*token.Index] = nil
				return
			}
			removePath(typed[*token.Index], tokens[1:])
		}
	}
}

func setPath(value any, tokens []pathToken, replacement any) {
	if len(tokens) == 0 {
		return
	}
	token := tokens[0]
	last := len(tokens) == 1
	switch typed := value.(type) {
	case map[string]any:
		if token.Wildcard {
			for _, item := range typed {
				setPath(item, tokens[1:], replacement)
			}
			return
		}
		if last {
			if _, ok := typed[token.Key]; ok {
				typed[token.Key] = replacement
			}
			return
		}
		setPath(typed[token.Key], tokens[1:], replacement)
	case []any:
		if token.Wildcard {
			for _, item := range typed {
				setPath(item, tokens[1:], replacement)
			}
			return
		}
		if token.Index != nil && *token.Index >= 0 && *token.Index < len(typed) {
			if last {
				typed[*token.Index] = replacement
				return
			}
			setPath(typed[*token.Index], tokens[1:], replacement)
		}
	}
}

func shapeOf(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := map[string]any{}
		for key, item := range typed {
			out[key] = shapeOf(item)
		}
		return out
	case []any:
		if len(typed) == 0 {
			return []any{}
		}
		return []any{shapeOf(typed[0])}
	default:
		return valueType(value)
	}
}

type retrySettings struct {
	maxAttempts int
	backoff     time.Duration
	retryable   map[string]bool
}

func retryPolicy(cfg config.RetryConfig) retrySettings {
	maxAttempts := cfg.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	backoff := time.Duration(0)
	if cfg.Backoff != "" {
		if parsed, err := time.ParseDuration(os.ExpandEnv(cfg.Backoff)); err == nil && parsed > 0 {
			backoff = parsed
		}
	}
	classes := cfg.RetryableFailureClasses
	if len(classes) == 0 {
		classes = []string{FailureClassNetwork, FailureClassTimeout}
	}
	return retrySettings{
		maxAttempts: maxAttempts,
		backoff:     backoff,
		retryable:   stringBoolSet(classes),
	}
}

func classifyRequestError(err error) string {
	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return FailureClassTimeout
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return FailureClassTimeout
	}
	return FailureClassNetwork
}

func setFailureClass(run *FixtureRunResult, class string) {
	if run.FailureClass == "" {
		run.FailureClass = class
	}
}

func fixtureMaxLatency(cfg *config.Config, fixture Fixture, suiteLatencyMs int) int {
	if fixture.MaxLatencyMs > 0 {
		return fixture.MaxLatencyMs
	}
	if suiteLatencyMs > 0 {
		return suiteLatencyMs
	}
	return cfg.Tests.MaxLatencyMs
}

func suiteMaxLatency(cfg *config.Config, suiteName string) int {
	if suiteName == "" {
		return 0
	}
	return cfg.Tests.Suites[suiteName].MaxLatencyMs
}

func timingSummary(results []FixtureRunResult) TimingSummary {
	var durations []int64
	for _, result := range results {
		if result.DurationMs > 0 {
			durations = append(durations, result.DurationMs)
		}
	}
	if len(durations) == 0 {
		return TimingSummary{}
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	return TimingSummary{
		MinMs: durations[0],
		MaxMs: durations[len(durations)-1],
		P50Ms: percentile(durations, 0.50),
		P95Ms: percentile(durations, 0.95),
	}
}

func percentile(values []int64, p float64) int64 {
	if len(values) == 0 {
		return 0
	}
	index := int(math.Ceil(float64(len(values))*p)) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func stringBoolSet(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out[value] = true
		}
	}
	return out
}

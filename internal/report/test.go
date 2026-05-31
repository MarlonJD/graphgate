package report

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/MarlonJD/graphgate/internal/core"
)

func RenderTestJSON(result core.TestRunResult) ([]byte, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func RenderTestMarkdown(result core.TestRunResult) []byte {
	var buf bytes.Buffer
	status := "PASS"
	if !result.OK {
		status = "FAIL"
	}
	fmt.Fprintf(&buf, "# GraphGate Test Report\n\n")
	fmt.Fprintf(&buf, "- Status: **%s**\n", status)
	fmt.Fprintf(&buf, "- Environment: `%s`\n", result.Environment)
	fmt.Fprintf(&buf, "- Endpoint: `%s`\n", result.Endpoint)
	if result.Selection.Suite != "" {
		fmt.Fprintf(&buf, "- Suite: `%s`\n", result.Selection.Suite)
	}
	if len(result.Selection.Tags) > 0 {
		fmt.Fprintf(&buf, "- Tags: `%s`\n", joinFailures(result.Selection.Tags))
	}
	if len(result.Selection.Exclude) > 0 {
		fmt.Fprintf(&buf, "- Exclude: `%s`\n", joinFailures(result.Selection.Exclude))
	}
	if result.FailureClass != "" {
		fmt.Fprintf(&buf, "- Failure class: `%s`\n", result.FailureClass)
	}
	fmt.Fprintf(&buf, "- Passed: %d\n", result.Passed)
	fmt.Fprintf(&buf, "- Failed: %d\n", result.Failed)
	if result.Timing.MaxMs > 0 {
		fmt.Fprintf(&buf, "- Timing: min %dms, p50 %dms, p95 %dms, max %dms\n", result.Timing.MinMs, result.Timing.P50Ms, result.Timing.P95Ms, result.Timing.MaxMs)
	}
	fmt.Fprintf(&buf, "\n")

	if len(result.Failures) > 0 {
		fmt.Fprintf(&buf, "## Run Failures\n\n")
		for _, failure := range result.Failures {
			fmt.Fprintf(&buf, "- %s\n", failure)
		}
		fmt.Fprintf(&buf, "\n")
	}

	if len(result.Readiness) > 0 {
		fmt.Fprintf(&buf, "## Readiness\n\n")
		fmt.Fprintf(&buf, "| Check | Result | Class | Message |\n")
		fmt.Fprintf(&buf, "| --- | --- | --- | --- |\n")
		for _, item := range result.Readiness {
			status := "PASS"
			if !item.Passed {
				status = "FAIL"
			}
			fmt.Fprintf(&buf, "| `%s` | **%s** | `%s` | %s |\n", escapeTable(item.Name), status, escapeTable(item.FailureClass), escapeTable(item.Message))
		}
		fmt.Fprintf(&buf, "\n")
	}

	if len(result.Results) == 0 {
		fmt.Fprintf(&buf, "No fixture results.\n")
		return buf.Bytes()
	}
	fmt.Fprintf(&buf, "| Fixture | Operation | Request operation | Tags | Status | Duration | Attempts | Result | Class | Failures |\n")
	fmt.Fprintf(&buf, "| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, item := range result.Results {
		rowStatus := "PASS"
		if !item.Passed {
			rowStatus = "FAIL"
		}
		requestOperation := item.RequestOperationName
		if requestOperation == "" {
			requestOperation = item.Operation
		}
		fmt.Fprintf(&buf, "| `%s` | `%s` | `%s` | `%s` | `%d` | `%dms` | `%d` | **%s** | `%s` | %s |\n", escapeTable(item.File), escapeTable(item.Operation), escapeTable(requestOperation), escapeTable(joinFailures(item.Tags)), item.Status, item.DurationMs, item.Attempts, rowStatus, escapeTable(item.FailureClass), escapeTable(joinFailures(item.Failures)))
	}
	return buf.Bytes()
}

func joinFailures(failures []string) string {
	if len(failures) == 0 {
		return ""
	}
	out := failures[0]
	for i := 1; i < len(failures); i++ {
		out += "; " + failures[i]
	}
	return out
}

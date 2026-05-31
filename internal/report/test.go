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
	fmt.Fprintf(&buf, "- Passed: %d\n", result.Passed)
	fmt.Fprintf(&buf, "- Failed: %d\n\n", result.Failed)

	if len(result.Results) == 0 {
		fmt.Fprintf(&buf, "No fixture results.\n")
		return buf.Bytes()
	}
	fmt.Fprintf(&buf, "| Fixture | Operation | Request operation | Status | Result | Failures |\n")
	fmt.Fprintf(&buf, "| --- | --- | --- | --- | --- | --- |\n")
	for _, item := range result.Results {
		rowStatus := "PASS"
		if !item.Passed {
			rowStatus = "FAIL"
		}
		requestOperation := item.RequestOperationName
		if requestOperation == "" {
			requestOperation = item.Operation
		}
		fmt.Fprintf(&buf, "| `%s` | `%s` | `%s` | `%d` | **%s** | %s |\n", escapeTable(item.File), escapeTable(item.Operation), escapeTable(requestOperation), item.Status, rowStatus, escapeTable(joinFailures(item.Failures)))
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

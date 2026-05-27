package report

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/MarlonJD/graphgate/internal/core"
)

func RenderDiffJSON(result core.DiffResult) ([]byte, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func RenderDiffMarkdown(result core.DiffResult) []byte {
	var buf bytes.Buffer
	status := "PASS"
	if !result.OK {
		status = "FAIL"
	}
	fmt.Fprintf(&buf, "# GraphGate Diff Report\n\n")
	fmt.Fprintf(&buf, "- Status: **%s**\n", status)
	fmt.Fprintf(&buf, "- Base schema: `%s`\n", result.BaseSchema)
	fmt.Fprintf(&buf, "- Current schema: `%s`\n", result.CurrentSchema)
	fmt.Fprintf(&buf, "- Breaking changes: %d\n", len(result.BreakingChanges))
	fmt.Fprintf(&buf, "- Impacted operations: %d\n\n", len(result.ImpactedOperations))

	fmt.Fprintf(&buf, "## Impacted Operations\n\n")
	if len(result.ImpactedOperations) == 0 {
		fmt.Fprintf(&buf, "No impacted operations found.\n\n")
	} else {
		fmt.Fprintf(&buf, "| Operation | File | Path | Change |\n")
		fmt.Fprintf(&buf, "| --- | --- | --- | --- |\n")
		for _, impact := range result.ImpactedOperations {
			fmt.Fprintf(&buf, "| `%s` | `%s` | `%s` | %s |\n", escapeTable(impact.Operation), escapeTable(impact.File), escapeTable(impact.Path), escapeTable(impact.Message))
		}
		fmt.Fprintf(&buf, "\n")
	}

	fmt.Fprintf(&buf, "## Breaking Changes\n\n")
	if len(result.BreakingChanges) == 0 {
		fmt.Fprintf(&buf, "No breaking changes found.\n")
		return buf.Bytes()
	}
	fmt.Fprintf(&buf, "| Kind | Type | Field | Message |\n")
	fmt.Fprintf(&buf, "| --- | --- | --- | --- |\n")
	for _, change := range result.BreakingChanges {
		fmt.Fprintf(&buf, "| `%s` | `%s` | `%s` | %s |\n", change.Kind, escapeTable(change.Type), escapeTable(change.Field), escapeTable(change.Message))
	}
	return buf.Bytes()
}

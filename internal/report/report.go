package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MarlonJD/graphgate/internal/core"
)

type Summary struct {
	OK             bool             `json:"ok"`
	Schema         string           `json:"schema"`
	OperationFiles []string         `json:"operationFiles"`
	Operations     []core.Operation `json:"operations"`
	Issues         []core.Issue     `json:"issues"`
	Manifest       core.Manifest    `json:"manifest"`
	ManifestOutput string           `json:"manifestOutput"`
}

func NewSummary(result core.ValidationResult, manifest core.Manifest, manifestOutput string) Summary {
	if result.OperationFiles == nil {
		result.OperationFiles = []string{}
	}
	if result.Operations == nil {
		result.Operations = []core.Operation{}
	}
	if result.Issues == nil {
		result.Issues = []core.Issue{}
	}
	if manifest.Operations == nil {
		manifest.Operations = []core.ManifestOperation{}
	}
	return Summary{
		OK:             result.OK(),
		Schema:         result.Schema,
		OperationFiles: result.OperationFiles,
		Operations:     result.Operations,
		Issues:         result.Issues,
		Manifest:       manifest,
		ManifestOutput: manifestOutput,
	}
}

func RenderJSON(summary Summary) ([]byte, error) {
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func RenderMarkdown(summary Summary) []byte {
	var buf bytes.Buffer
	status := "PASS"
	if !summary.OK {
		status = "FAIL"
	}

	fmt.Fprintf(&buf, "# GraphGate Report\n\n")
	fmt.Fprintf(&buf, "- Status: **%s**\n", status)
	fmt.Fprintf(&buf, "- Schema: `%s`\n", summary.Schema)
	fmt.Fprintf(&buf, "- Operation files: %d\n", len(summary.OperationFiles))
	fmt.Fprintf(&buf, "- Operations: %d\n", len(summary.Operations))
	fmt.Fprintf(&buf, "- Manifest output: `%s`\n\n", summary.ManifestOutput)

	fmt.Fprintf(&buf, "## Issues\n\n")
	if len(summary.Issues) == 0 {
		fmt.Fprintf(&buf, "No issues found.\n\n")
	} else {
		fmt.Fprintf(&buf, "| Code | Location | Operation | Message |\n")
		fmt.Fprintf(&buf, "| --- | --- | --- | --- |\n")
		for _, issue := range summary.Issues {
			fmt.Fprintf(
				&buf,
				"| `%s` | `%s` | `%s` | %s |\n",
				issue.Code,
				issueLocation(issue),
				escapeTable(issue.Operation),
				escapeTable(issue.Message),
			)
		}
		fmt.Fprintf(&buf, "\n")
	}

	fmt.Fprintf(&buf, "## Operations\n\n")
	if len(summary.Operations) == 0 {
		fmt.Fprintf(&buf, "No operations found.\n")
		return buf.Bytes()
	}

	fmt.Fprintf(&buf, "| Operation | ID | File |\n")
	fmt.Fprintf(&buf, "| --- | --- | --- |\n")
	for _, operation := range summary.Operations {
		fmt.Fprintf(
			&buf,
			"| `%s` | `%s` | `%s` |\n",
			escapeTable(operation.Name),
			shortID(operation.ID),
			escapeTable(operation.File),
		)
	}
	return buf.Bytes()
}

func issueLocation(issue core.Issue) string {
	location := issue.File
	if issue.Line > 0 {
		location = fmt.Sprintf("%s:%d", location, issue.Line)
		if issue.Column > 0 {
			location = fmt.Sprintf("%s:%d", location, issue.Column)
		}
	}
	return location
}

func shortID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}

func escapeTable(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}

package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MarlonJD/graphgate/internal/config"
)

func TestValidateProjectValidOperation(t *testing.T) {
	cfg := loadTestConfig(t, map[string]string{
		"schema.graphql":               "type Query { viewer: User! }\ntype User { id: ID!, name: String! }\n",
		"operations/GetViewer.graphql": "query GetViewer { viewer { id name } }\n",
	})

	result, err := ValidateProject(cfg)
	if err != nil {
		t.Fatalf("ValidateProject() error = %v", err)
	}
	if !result.OK() {
		t.Fatalf("ValidateProject() issues = %#v", result.Issues)
	}
	if len(result.Operations) != 1 {
		t.Fatalf("operations len = %d", len(result.Operations))
	}
	if result.Operations[0].ID == "" || result.Operations[0].SHA256 != result.Operations[0].ID {
		t.Fatalf("operation ID not populated: %#v", result.Operations[0])
	}
}

func TestValidateProjectInvalidSchema(t *testing.T) {
	cfg := loadTestConfig(t, map[string]string{
		"schema.graphql":               "type Query { viewer: }\n",
		"operations/GetViewer.graphql": "query GetViewer { viewer }\n",
	})

	result, err := ValidateProject(cfg)
	if err != nil {
		t.Fatalf("ValidateProject() error = %v", err)
	}
	if !result.HasSchemaIssues() {
		t.Fatalf("issues = %#v, want schema issue", result.Issues)
	}
}

func TestValidateProjectInvalidField(t *testing.T) {
	cfg := loadTestConfig(t, map[string]string{
		"schema.graphql":               "type Query { viewer: String! }\n",
		"operations/GetViewer.graphql": "query GetViewer { missing }\n",
	})

	result, err := ValidateProject(cfg)
	if err != nil {
		t.Fatalf("ValidateProject() error = %v", err)
	}
	if !hasIssue(result, CodeInvalidOperation) {
		t.Fatalf("issues = %#v, want invalid operation", result.Issues)
	}
}

func TestValidateProjectRejectsAnonymousOperation(t *testing.T) {
	cfg := loadTestConfig(t, map[string]string{
		"schema.graphql":               "type Query { viewer: String! }\n",
		"operations/GetViewer.graphql": "{ viewer }\n",
	})

	result, err := ValidateProject(cfg)
	if err != nil {
		t.Fatalf("ValidateProject() error = %v", err)
	}
	if !hasIssue(result, CodeAnonymousOperation) {
		t.Fatalf("issues = %#v, want anonymous operation", result.Issues)
	}
}

func TestValidateProjectRejectsDuplicateOperationNames(t *testing.T) {
	cfg := loadTestConfig(t, map[string]string{
		"schema.graphql":           "type Query { viewer: String! }\n",
		"operations/a.graphql":     "query Viewer { viewer }\n",
		"operations/nested/b.gql":  "query Viewer { viewer }\n",
		"operations/nested/c.txt":  "query NotMatched { viewer }\n",
		"operations/nested/d.json": "{}\n",
	})

	result, err := ValidateProject(cfg)
	if err != nil {
		t.Fatalf("ValidateProject() error = %v", err)
	}
	if !hasIssue(result, CodeDuplicateOperationName) {
		t.Fatalf("issues = %#v, want duplicate operation name", result.Issues)
	}
}

func TestOperationIDIgnoresFormattingWhitespace(t *testing.T) {
	left := validateSingleOperation(t, "query GetViewer { viewer { id name } }\n")
	right := validateSingleOperation(t, "query GetViewer {\n  viewer {\n    id\n    name\n  }\n}\n")

	if left.Operations[0].ID != right.Operations[0].ID {
		t.Fatalf("IDs differ:\nleft=%s\nright=%s", left.Operations[0].ID, right.Operations[0].ID)
	}
}

func validateSingleOperation(t *testing.T, operation string) ValidationResult {
	t.Helper()
	cfg := loadTestConfig(t, map[string]string{
		"schema.graphql":               "type Query { viewer: User! }\ntype User { id: ID!, name: String! }\n",
		"operations/GetViewer.graphql": operation,
	})
	result, err := ValidateProject(cfg)
	if err != nil {
		t.Fatalf("ValidateProject() error = %v", err)
	}
	if !result.OK() {
		t.Fatalf("issues = %#v", result.Issues)
	}
	return result
}

func loadTestConfig(t *testing.T, files map[string]string) *config.Config {
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

	cfgData := []byte("schema: ./schema.graphql\noperations:\n  - ./operations/**/*.graphql\n  - ./operations/**/*.gql\nmanifest:\n  format: graphgate\n  output: ./graphgate.manifest.json\n")
	cfgPath := filepath.Join(dir, config.DefaultConfigPath)
	if err := os.WriteFile(cfgPath, cfgData, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	return cfg
}

func hasIssue(result ValidationResult, code IssueCode) bool {
	for _, issue := range result.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

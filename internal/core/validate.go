package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/MarlonJD/graphgate/internal/config"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/vektah/gqlparser/v2/parser"
	"github.com/vektah/gqlparser/v2/validator"
)

func ValidateProject(cfg *config.Config) (ValidationResult, error) {
	result := ValidationResult{
		Schema:         cfg.RelativePath(cfg.ResolvePath(cfg.Schema)),
		OperationFiles: []string{},
		Operations:     []Operation{},
		Issues:         []Issue{},
	}

	schema, schemaIssues, err := loadSchema(cfg)
	if err != nil {
		return result, err
	}
	if len(schemaIssues) > 0 {
		result.Issues = append(result.Issues, schemaIssues...)
		return result, nil
	}

	files, err := cfg.OperationFiles()
	if err != nil {
		return result, err
	}
	for _, file := range files {
		result.OperationFiles = append(result.OperationFiles, cfg.RelativePath(file))
	}

	seenNames := map[string]string{}
	for _, file := range files {
		operations, issues := validateOperationFile(cfg, schema, file, seenNames)
		result.Operations = append(result.Operations, operations...)
		result.Issues = append(result.Issues, issues...)
	}

	sort.Slice(result.Operations, func(i, j int) bool {
		if result.Operations[i].Name == result.Operations[j].Name {
			return result.Operations[i].File < result.Operations[j].File
		}
		return result.Operations[i].Name < result.Operations[j].Name
	})

	return result, nil
}

func validateOperationFile(cfg *config.Config, schema *ast.Schema, file string, seenNames map[string]string) ([]Operation, []Issue) {
	relFile := cfg.RelativePath(file)
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, []Issue{{
			Code:    CodeInvalidOperation,
			Message: fmt.Sprintf("read operation file: %v", err),
			File:    relFile,
		}}
	}

	doc, err := parser.ParseQuery(&ast.Source{
		Name:  relFile,
		Input: string(data),
	})
	if err != nil {
		return nil, issuesFromError(CodeInvalidOperation, relFile, "", err)
	}

	var issues []Issue
	for _, op := range doc.Operations {
		if op.Name == "" {
			issues = append(issues, Issue{
				Code:    CodeAnonymousOperation,
				Message: "anonymous operations are not allowed because persisted manifests require stable operation names",
				File:    relFile,
			})
			continue
		}
		if previousFile, ok := seenNames[op.Name]; ok {
			issues = append(issues, Issue{
				Code:      CodeDuplicateOperationName,
				Message:   fmt.Sprintf("operation %q is already defined in %s", op.Name, previousFile),
				File:      relFile,
				Operation: op.Name,
			})
			continue
		}
		seenNames[op.Name] = relFile
	}

	if gqlErrs := validator.Validate(schema, doc); len(gqlErrs) > 0 {
		issues = append(issues, issuesFromGraphQLErrors(CodeInvalidOperation, relFile, "", gqlErrs)...)
	}

	var operations []Operation
	for _, op := range doc.Operations {
		if op.Name == "" {
			continue
		}
		normalized := normalizeOperation(op, doc.Fragments)
		id := operationID(normalized)
		operations = append(operations, Operation{
			Name:       op.Name,
			ID:         id,
			SHA256:     id,
			File:       relFile,
			Normalized: normalized,
		})
	}

	return operations, issues
}

func normalizeOperation(op *ast.OperationDefinition, fragments ast.FragmentDefinitionList) string {
	var buf bytes.Buffer
	doc := &ast.QueryDocument{
		Operations: ast.OperationList{op},
		Fragments:  fragments,
	}
	formatter.NewFormatter(&buf, formatter.WithCompacted()).FormatQueryDocument(doc)
	return buf.String()
}

func operationID(normalized string) string {
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func issuesFromError(code IssueCode, fallbackFile string, operation string, err error) []Issue {
	var gqlErr *gqlerror.Error
	if errors.As(err, &gqlErr) {
		return issuesFromGraphQLErrors(code, fallbackFile, operation, gqlerror.List{gqlErr})
	}
	return []Issue{{
		Code:      code,
		Message:   err.Error(),
		File:      fallbackFile,
		Operation: operation,
	}}
}

func issuesFromGraphQLErrors(code IssueCode, fallbackFile string, operation string, errs gqlerror.List) []Issue {
	issues := make([]Issue, 0, len(errs))
	for _, gqlErr := range errs {
		file := fallbackFile
		if gqlErr.Extensions != nil {
			if extensionFile, ok := gqlErr.Extensions["file"].(string); ok && extensionFile != "" {
				file = extensionFile
			}
		}
		issue := Issue{
			Code:      code,
			Message:   gqlErr.Message,
			File:      file,
			Operation: operation,
		}
		if len(gqlErr.Locations) > 0 {
			issue.Line = gqlErr.Locations[0].Line
			issue.Column = gqlErr.Locations[0].Column
		}
		issues = append(issues, issue)
	}
	return issues
}

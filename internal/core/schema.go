package core

import (
	"fmt"
	"os"

	"github.com/MarlonJD/graphgate/internal/config"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

func loadSchema(cfg *config.Config) (*ast.Schema, []Issue, error) {
	path := cfg.ResolvePath(cfg.Schema)
	return LoadSchemaFile(path, cfg.RelativePath(path))
}

func LoadSchemaFile(path string, displayName string) (*ast.Schema, []Issue, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read schema %q: %w", displayName, err)
	}

	schema, err := gqlparser.LoadSchema(&ast.Source{
		Name:  displayName,
		Input: string(data),
	})
	if err == nil {
		return schema, nil, nil
	}

	return nil, issuesFromError(CodeInvalidSchema, displayName, "", err), nil
}

type SchemaSummary struct {
	Types []SchemaType `json:"types"`
}

type SchemaType struct {
	Name   string        `json:"name"`
	Kind   string        `json:"kind"`
	Fields []SchemaField `json:"fields,omitempty"`
}

type SchemaField struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func SummarizeSchema(cfg *config.Config) (SchemaSummary, []Issue, error) {
	schema, issues, err := loadSchema(cfg)
	if err != nil || len(issues) > 0 {
		return SchemaSummary{}, issues, err
	}
	return summarizeSchema(schema), nil, nil
}

func summarizeSchema(schema *ast.Schema) SchemaSummary {
	var summary SchemaSummary
	for _, def := range sortedDefinitions(schema) {
		if skipSchemaType(def) {
			continue
		}
		item := SchemaType{
			Name: def.Name,
			Kind: string(def.Kind),
		}
		for _, field := range def.Fields {
			if len(field.Name) >= 2 && field.Name[:2] == "__" {
				continue
			}
			item.Fields = append(item.Fields, SchemaField{
				Name: field.Name,
				Type: field.Type.String(),
			})
		}
		summary.Types = append(summary.Types, item)
	}
	return summary
}

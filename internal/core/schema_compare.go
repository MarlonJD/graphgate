package core

import (
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

func sortedDefinitions(schema *ast.Schema) []*ast.Definition {
	defs := make([]*ast.Definition, 0, len(schema.Types))
	for _, def := range schema.Types {
		defs = append(defs, def)
	}
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].Name < defs[j].Name
	})
	return defs
}

func skipSchemaType(def *ast.Definition) bool {
	return def == nil || def.BuiltIn || strings.HasPrefix(def.Name, "__")
}

func fieldMap(def *ast.Definition) map[string]*ast.FieldDefinition {
	fields := map[string]*ast.FieldDefinition{}
	if def == nil {
		return fields
	}
	for _, field := range def.Fields {
		fields[field.Name] = field
	}
	return fields
}

func argMap(field *ast.FieldDefinition) map[string]*ast.ArgumentDefinition {
	args := map[string]*ast.ArgumentDefinition{}
	if field == nil {
		return args
	}
	for _, arg := range field.Arguments {
		args[arg.Name] = arg
	}
	return args
}

func enumValueMap(def *ast.Definition) map[string]struct{} {
	values := map[string]struct{}{}
	if def == nil {
		return values
	}
	for _, value := range def.EnumValues {
		values[value.Name] = struct{}{}
	}
	return values
}

func stringSet(values []string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func isRequiredInput(t *ast.Type, hasDefault bool) bool {
	return t != nil && t.NonNull && !hasDefault
}

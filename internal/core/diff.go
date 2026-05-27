package core

import (
	"fmt"
	"os"
	"sort"

	"github.com/MarlonJD/graphgate/internal/config"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

type ChangeKind string

const (
	ChangeTypeRemoved             ChangeKind = "type_removed"
	ChangeFieldRemoved            ChangeKind = "field_removed"
	ChangeFieldTypeChanged        ChangeKind = "field_type_changed"
	ChangeArgumentRemoved         ChangeKind = "argument_removed"
	ChangeRequiredArgAdded        ChangeKind = "required_argument_added"
	ChangeEnumValueRemoved        ChangeKind = "enum_value_removed"
	ChangeUnionMemberRemoved      ChangeKind = "union_member_removed"
	ChangeInputFieldRemoved       ChangeKind = "input_field_removed"
	ChangeInputFieldTypeChanged   ChangeKind = "input_field_type_changed"
	ChangeRequiredInputFieldAdded ChangeKind = "required_input_field_added"
)

type SchemaChange struct {
	Kind     ChangeKind `json:"kind"`
	Message  string     `json:"message"`
	Type     string     `json:"type,omitempty"`
	Field    string     `json:"field,omitempty"`
	Argument string     `json:"argument,omitempty"`
	From     string     `json:"from,omitempty"`
	To       string     `json:"to,omitempty"`
}

type ImpactedOperation struct {
	Operation string     `json:"operation"`
	File      string     `json:"file"`
	Kind      ChangeKind `json:"kind"`
	Message   string     `json:"message"`
	Type      string     `json:"type,omitempty"`
	Field     string     `json:"field,omitempty"`
	Path      string     `json:"path,omitempty"`
}

type DiffResult struct {
	OK                 bool                `json:"ok"`
	BaseSchema         string              `json:"baseSchema"`
	CurrentSchema      string              `json:"currentSchema"`
	BreakingChanges    []SchemaChange      `json:"breakingChanges"`
	ImpactedOperations []ImpactedOperation `json:"impactedOperations"`
}

func DiffSchemas(cfg *config.Config, baseSchemaPath string) (DiffResult, error) {
	basePath := cfg.ResolvePath(baseSchemaPath)
	currentPath := cfg.ResolvePath(cfg.Schema)
	result := DiffResult{
		BaseSchema:         cfg.RelativePath(basePath),
		CurrentSchema:      cfg.RelativePath(currentPath),
		BreakingChanges:    []SchemaChange{},
		ImpactedOperations: []ImpactedOperation{},
	}

	baseSchema, issues, err := LoadSchemaFile(basePath, result.BaseSchema)
	if err != nil {
		return result, err
	}
	if len(issues) > 0 {
		return result, fmt.Errorf("base schema is invalid: %s", issues[0].Message)
	}
	currentSchema, issues, err := loadSchema(cfg)
	if err != nil {
		return result, err
	}
	if len(issues) > 0 {
		return result, fmt.Errorf("current schema is invalid: %s", issues[0].Message)
	}

	result.BreakingChanges = compareSchemas(baseSchema, currentSchema)
	usages, err := collectOperationUsages(cfg, baseSchema)
	if err != nil {
		return result, err
	}
	result.ImpactedOperations = impactedOperations(result.BreakingChanges, usages)
	if result.BreakingChanges == nil {
		result.BreakingChanges = []SchemaChange{}
	}
	if result.ImpactedOperations == nil {
		result.ImpactedOperations = []ImpactedOperation{}
	}
	result.OK = len(result.ImpactedOperations) == 0
	return result, nil
}

func compareSchemas(baseSchema *ast.Schema, currentSchema *ast.Schema) []SchemaChange {
	var changes []SchemaChange
	for _, baseDef := range sortedDefinitions(baseSchema) {
		if skipSchemaType(baseDef) {
			continue
		}
		currentDef := currentSchema.Types[baseDef.Name]
		if currentDef == nil {
			changes = append(changes, SchemaChange{
				Kind:    ChangeTypeRemoved,
				Message: fmt.Sprintf("type %s was removed", baseDef.Name),
				Type:    baseDef.Name,
			})
			continue
		}

		if baseDef.Kind == ast.Object || baseDef.Kind == ast.Interface {
			changes = append(changes, compareOutputFields(baseDef, currentDef)...)
		}
		if baseDef.Kind == ast.InputObject {
			changes = append(changes, compareInputFields(baseDef, currentDef)...)
		}
		if baseDef.Kind == ast.Enum {
			changes = append(changes, compareEnumValues(baseDef, currentDef)...)
		}
		if baseDef.Kind == ast.Union {
			changes = append(changes, compareUnionMembers(baseDef, currentDef)...)
		}
	}

	for _, currentDef := range sortedDefinitions(currentSchema) {
		if skipSchemaType(currentDef) || currentDef.Kind != ast.InputObject {
			continue
		}
		baseDef := baseSchema.Types[currentDef.Name]
		if baseDef == nil {
			continue
		}
		baseFields := fieldMap(baseDef)
		for _, currentField := range currentDef.Fields {
			if baseFields[currentField.Name] == nil && isRequiredInput(currentField.Type, currentField.DefaultValue != nil) {
				changes = append(changes, SchemaChange{
					Kind:    ChangeRequiredInputFieldAdded,
					Message: fmt.Sprintf("required input field %s.%s was added", currentDef.Name, currentField.Name),
					Type:    currentDef.Name,
					Field:   currentField.Name,
					To:      currentField.Type.String(),
				})
			}
		}
	}

	sort.Slice(changes, func(i, j int) bool {
		if changes[i].Type == changes[j].Type {
			if changes[i].Field == changes[j].Field {
				return changes[i].Kind < changes[j].Kind
			}
			return changes[i].Field < changes[j].Field
		}
		return changes[i].Type < changes[j].Type
	})
	return changes
}

func compareOutputFields(baseDef *ast.Definition, currentDef *ast.Definition) []SchemaChange {
	var changes []SchemaChange
	currentFields := fieldMap(currentDef)
	for _, baseField := range baseDef.Fields {
		currentField := currentFields[baseField.Name]
		if currentField == nil {
			changes = append(changes, SchemaChange{
				Kind:    ChangeFieldRemoved,
				Message: fmt.Sprintf("field %s.%s was removed", baseDef.Name, baseField.Name),
				Type:    baseDef.Name,
				Field:   baseField.Name,
				From:    baseField.Type.String(),
			})
			continue
		}
		if baseField.Type.String() != currentField.Type.String() {
			changes = append(changes, SchemaChange{
				Kind:    ChangeFieldTypeChanged,
				Message: fmt.Sprintf("field %s.%s changed type from %s to %s", baseDef.Name, baseField.Name, baseField.Type.String(), currentField.Type.String()),
				Type:    baseDef.Name,
				Field:   baseField.Name,
				From:    baseField.Type.String(),
				To:      currentField.Type.String(),
			})
		}
		changes = append(changes, compareFieldArguments(baseDef, baseField, currentField)...)
	}
	return changes
}

func compareFieldArguments(parent *ast.Definition, baseField *ast.FieldDefinition, currentField *ast.FieldDefinition) []SchemaChange {
	var changes []SchemaChange
	currentArgs := argMap(currentField)
	for _, baseArg := range baseField.Arguments {
		if currentArgs[baseArg.Name] == nil {
			changes = append(changes, SchemaChange{
				Kind:     ChangeArgumentRemoved,
				Message:  fmt.Sprintf("argument %s.%s(%s:) was removed", parent.Name, baseField.Name, baseArg.Name),
				Type:     parent.Name,
				Field:    baseField.Name,
				Argument: baseArg.Name,
				From:     baseArg.Type.String(),
			})
		}
	}

	baseArgs := argMap(baseField)
	for _, currentArg := range currentField.Arguments {
		if baseArgs[currentArg.Name] == nil && isRequiredInput(currentArg.Type, currentArg.DefaultValue != nil) {
			changes = append(changes, SchemaChange{
				Kind:     ChangeRequiredArgAdded,
				Message:  fmt.Sprintf("required argument %s.%s(%s:) was added", parent.Name, baseField.Name, currentArg.Name),
				Type:     parent.Name,
				Field:    baseField.Name,
				Argument: currentArg.Name,
				To:       currentArg.Type.String(),
			})
		}
	}
	return changes
}

func compareInputFields(baseDef *ast.Definition, currentDef *ast.Definition) []SchemaChange {
	var changes []SchemaChange
	currentFields := fieldMap(currentDef)
	for _, baseField := range baseDef.Fields {
		currentField := currentFields[baseField.Name]
		if currentField == nil {
			changes = append(changes, SchemaChange{
				Kind:    ChangeInputFieldRemoved,
				Message: fmt.Sprintf("input field %s.%s was removed", baseDef.Name, baseField.Name),
				Type:    baseDef.Name,
				Field:   baseField.Name,
				From:    baseField.Type.String(),
			})
			continue
		}
		if baseField.Type.String() != currentField.Type.String() {
			changes = append(changes, SchemaChange{
				Kind:    ChangeInputFieldTypeChanged,
				Message: fmt.Sprintf("input field %s.%s changed type from %s to %s", baseDef.Name, baseField.Name, baseField.Type.String(), currentField.Type.String()),
				Type:    baseDef.Name,
				Field:   baseField.Name,
				From:    baseField.Type.String(),
				To:      currentField.Type.String(),
			})
		}
	}
	return changes
}

func compareEnumValues(baseDef *ast.Definition, currentDef *ast.Definition) []SchemaChange {
	var changes []SchemaChange
	currentValues := enumValueMap(currentDef)
	for _, value := range baseDef.EnumValues {
		if _, ok := currentValues[value.Name]; !ok {
			changes = append(changes, SchemaChange{
				Kind:    ChangeEnumValueRemoved,
				Message: fmt.Sprintf("enum value %s.%s was removed", baseDef.Name, value.Name),
				Type:    baseDef.Name,
				Field:   value.Name,
			})
		}
	}
	return changes
}

func compareUnionMembers(baseDef *ast.Definition, currentDef *ast.Definition) []SchemaChange {
	var changes []SchemaChange
	currentMembers := stringSet(currentDef.Types)
	for _, member := range baseDef.Types {
		if _, ok := currentMembers[member]; !ok {
			changes = append(changes, SchemaChange{
				Kind:    ChangeUnionMemberRemoved,
				Message: fmt.Sprintf("union member %s.%s was removed", baseDef.Name, member),
				Type:    baseDef.Name,
				Field:   member,
			})
		}
	}
	return changes
}

type operationUsage struct {
	Operation string
	File      string
	Fields    map[string]fieldUsage
}

type fieldUsage struct {
	Type  string
	Field string
	Path  string
}

func collectOperationUsages(cfg *config.Config, schema *ast.Schema) ([]operationUsage, error) {
	files, err := cfg.OperationFiles()
	if err != nil {
		return nil, err
	}
	var usages []operationUsage
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		doc, errs := gqlparser.LoadQuery(schema, string(data))
		if len(errs) > 0 {
			return nil, fmt.Errorf("operation %s is invalid against base schema: %s", cfg.RelativePath(file), errs.Error())
		}
		for _, op := range doc.Operations {
			if op.Name == "" {
				continue
			}
			usage := operationUsage{
				Operation: op.Name,
				File:      cfg.RelativePath(file),
				Fields:    map[string]fieldUsage{},
			}
			root := operationRoot(schema, op.Operation)
			collectSelectionFields(schema, doc, root, op.SelectionSet, op.Name, &usage)
			usages = append(usages, usage)
		}
	}
	return usages, nil
}

func operationRoot(schema *ast.Schema, op ast.Operation) *ast.Definition {
	switch op {
	case ast.Mutation:
		return schema.Mutation
	case ast.Subscription:
		return schema.Subscription
	default:
		return schema.Query
	}
}

func collectSelectionFields(schema *ast.Schema, doc *ast.QueryDocument, parent *ast.Definition, selections ast.SelectionSet, path string, usage *operationUsage) {
	if parent == nil {
		return
	}
	for _, selection := range selections {
		switch sel := selection.(type) {
		case *ast.Field:
			if sel.Name == "__typename" {
				continue
			}
			def := parent.Fields.ForName(sel.Name)
			if def == nil {
				continue
			}
			fieldPath := path + "." + sel.Name
			key := parent.Name + "." + sel.Name
			usage.Fields[key] = fieldUsage{Type: parent.Name, Field: sel.Name, Path: fieldPath}
			child := schema.Types[def.Type.Name()]
			collectSelectionFields(schema, doc, child, sel.SelectionSet, fieldPath, usage)
		case *ast.FragmentSpread:
			fragment := doc.Fragments.ForName(sel.Name)
			if fragment == nil {
				continue
			}
			fragmentType := parent
			if fragment.TypeCondition != "" {
				fragmentType = schema.Types[fragment.TypeCondition]
			}
			collectSelectionFields(schema, doc, fragmentType, fragment.SelectionSet, path+"."+sel.Name, usage)
		case *ast.InlineFragment:
			fragmentType := parent
			if sel.TypeCondition != "" {
				fragmentType = schema.Types[sel.TypeCondition]
			}
			collectSelectionFields(schema, doc, fragmentType, sel.SelectionSet, path, usage)
		}
	}
}

func impactedOperations(changes []SchemaChange, usages []operationUsage) []ImpactedOperation {
	var impacted []ImpactedOperation
	seen := map[string]struct{}{}
	for _, change := range changes {
		if change.Field == "" {
			continue
		}
		key := change.Type + "." + change.Field
		for _, usage := range usages {
			fieldUsage, ok := usage.Fields[key]
			if !ok {
				continue
			}
			impactKey := usage.Operation + "|" + usage.File + "|" + string(change.Kind) + "|" + key
			if _, exists := seen[impactKey]; exists {
				continue
			}
			seen[impactKey] = struct{}{}
			impacted = append(impacted, ImpactedOperation{
				Operation: usage.Operation,
				File:      usage.File,
				Kind:      change.Kind,
				Message:   change.Message,
				Type:      change.Type,
				Field:     change.Field,
				Path:      fieldUsage.Path,
			})
		}
	}
	sort.Slice(impacted, func(i, j int) bool {
		if impacted[i].Operation == impacted[j].Operation {
			return impacted[i].Message < impacted[j].Message
		}
		return impacted[i].Operation < impacted[j].Operation
	})
	return impacted
}

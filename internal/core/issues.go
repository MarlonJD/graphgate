package core

type IssueCode string

const (
	CodeInvalidSchema          IssueCode = "invalid_schema"
	CodeInvalidOperation       IssueCode = "invalid_operation"
	CodeAnonymousOperation     IssueCode = "anonymous_operation"
	CodeDuplicateOperationName IssueCode = "duplicate_operation_name"
)

type Issue struct {
	Code      IssueCode `json:"code"`
	Message   string    `json:"message"`
	File      string    `json:"file,omitempty"`
	Operation string    `json:"operation,omitempty"`
	Line      int       `json:"line,omitempty"`
	Column    int       `json:"column,omitempty"`
}

type Operation struct {
	Name             string   `json:"name"`
	ID               string   `json:"id"`
	SHA256           string   `json:"sha256"`
	File             string   `json:"file"`
	DeprecatedFields []string `json:"deprecatedFields,omitempty"`
	Normalized       string   `json:"-"`
}

type ValidationResult struct {
	Schema         string      `json:"schema"`
	OperationFiles []string    `json:"operationFiles"`
	Operations     []Operation `json:"operations"`
	Issues         []Issue     `json:"issues"`
}

func (r ValidationResult) OK() bool {
	return len(r.Issues) == 0
}

func (r ValidationResult) HasSchemaIssues() bool {
	for _, issue := range r.Issues {
		if issue.Code == CodeInvalidSchema {
			return true
		}
	}
	return false
}

func (r ValidationResult) HasOperationIssues() bool {
	for _, issue := range r.Issues {
		if issue.Code != CodeInvalidSchema {
			return true
		}
	}
	return false
}

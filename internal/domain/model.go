package domain

import "time"

const (
	RunStatusRunning   = "running"
	RunStatusSucceeded = "succeeded"
	RunStatusFailed    = "failed"
)

const (
	NodeStatusPending   = "pending"
	NodeStatusRunning   = "running"
	NodeStatusSucceeded = "succeeded"
	NodeStatusFailed    = "failed"
	NodeStatusSkipped   = "skipped"
)

type VariableDefinition struct {
	Name        string `yaml:"name" json:"name"`
	Required    bool   `yaml:"required" json:"required"`
	Default     string `yaml:"default,omitempty" json:"default,omitempty"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

type ToolDefinition struct {
	Name        string               `yaml:"-" json:"name"`
	Category    string               `yaml:"category" json:"category"`
	Description string               `yaml:"description,omitempty" json:"description,omitempty"`
	CmdTemplate string               `yaml:"command_template,omitempty" json:"command_template,omitempty"`
	Parser      string               `yaml:"parser,omitempty" json:"parser,omitempty"`
	Variables   []VariableDefinition `yaml:"variables,omitempty" json:"variables,omitempty"`
	PrivateVars []VariableDefinition `yaml:"private_variables,omitempty" json:"private_variables,omitempty"`
	Metadata    map[string]any       `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

type ToolCatalog struct {
	Tools map[string]ToolDefinition `yaml:"tools"`
}

type ParserRuleSet struct {
	Tool  string       `yaml:"tool" json:"tool"`
	Rules []ParserRule `yaml:"rules" json:"rules"`
}

type ParserRule struct {
	Name   string `yaml:"name" json:"name"`
	Regex  string `yaml:"regex" json:"regex"`
	SaveTo string `yaml:"save_to" json:"save_to"`
}

type Workflow struct {
	Name        string         `yaml:"name" json:"name"`
	Description string         `yaml:"description,omitempty" json:"description,omitempty"`
	GlobalVars  map[string]any `yaml:"global_vars,omitempty" json:"global_vars,omitempty"`
	Nodes       []WorkflowNode `yaml:"nodes" json:"nodes"`
	Edges       []WorkflowEdge `yaml:"edges,omitempty" json:"edges,omitempty"`
}

type WorkflowNode struct {
	ID       string         `yaml:"id" json:"id"`
	Tool     string         `yaml:"tool" json:"tool"`
	Executor string         `yaml:"executor" json:"executor"`
	Inputs   map[string]any `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Mock     MockConfig     `yaml:"mock,omitempty" json:"mock,omitempty"`
}

type MockConfig struct {
	StdoutFile string         `yaml:"stdout_file,omitempty" json:"stdout_file,omitempty"`
	Metadata   map[string]any `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

type WorkflowEdge struct {
	From      string `yaml:"from" json:"from"`
	To        string `yaml:"to" json:"to"`
	Condition string `yaml:"condition,omitempty" json:"condition,omitempty"`
}

type ExecutionResult struct {
	Stdout   string         `json:"stdout"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type ParsedRecord struct {
	Rule   string            `json:"rule"`
	Path   string            `json:"path"`
	Fields map[string]string `json:"fields"`
}

type RunStats struct {
	TotalNodes     int `json:"total_nodes"`
	ExecutedNodes  int `json:"executed_nodes"`
	SucceededNodes int `json:"succeeded_nodes"`
	FailedNodes    int `json:"failed_nodes"`
	SkippedNodes   int `json:"skipped_nodes"`
}

type NodeRunResult struct {
	NodeID          string         `json:"node_id"`
	Tool            string         `json:"tool"`
	Executor        string         `json:"executor"`
	Status          string         `json:"status"`
	RenderedCommand string         `json:"rendered_command,omitempty"`
	Inputs          map[string]any `json:"inputs,omitempty"`
	Stdout          string         `json:"stdout,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	Records         []ParsedRecord `json:"records,omitempty"`
	RecordCount     int            `json:"record_count"`
	Error           string         `json:"error,omitempty"`
	SkipReason      string         `json:"skip_reason,omitempty"`
	DurationMillis  int64          `json:"duration_millis"`
	StartedAt       time.Time      `json:"started_at"`
	FinishedAt      time.Time      `json:"finished_at"`
}

type RunSummary struct {
	Workflow       string                   `json:"workflow"`
	Status         string                   `json:"status"`
	Error          string                   `json:"error,omitempty"`
	StartedAt      time.Time                `json:"started_at"`
	FinishedAt     time.Time                `json:"finished_at"`
	Vars           map[string]any           `json:"vars"`
	Assets         map[string]any           `json:"assets"`
	NodeResults    map[string]NodeRunResult `json:"node_results"`
	ExecutionOrder []string                 `json:"execution_order"`
	Stats          RunStats                 `json:"stats"`
}

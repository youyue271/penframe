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
	ID              string         `yaml:"id" json:"id"`
	Tool            string         `yaml:"tool" json:"tool"`
	Executor        string         `yaml:"executor" json:"executor"`
	Shell           string         `yaml:"shell,omitempty" json:"shell,omitempty"`
	Env             map[string]any `yaml:"env,omitempty" json:"env,omitempty"`
	TimeoutSeconds  int            `yaml:"timeout_seconds,omitempty" json:"timeout_seconds,omitempty"`
	ContinueOnError bool           `yaml:"continue_on_error,omitempty" json:"continue_on_error,omitempty"`
	Inputs          map[string]any `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Mock            MockConfig     `yaml:"mock,omitempty" json:"mock,omitempty"`
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

// Asset graph models for layered scanning.

type AssetHost struct {
	ID       string      `json:"id"`
	IP       string      `json:"ip"`
	Hostname string      `json:"hostname,omitempty"`
	Ports    []AssetPort `json:"ports,omitempty"`
	Status   string      `json:"status"` // alive/unknown
	Source   string      `json:"source,omitempty"`
}

type AssetPort struct {
	ID       string      `json:"id"`
	HostID   string      `json:"host_id"`
	Port     int         `json:"port"`
	Protocol string      `json:"protocol"`
	Service  string      `json:"service,omitempty"`
	Banner   string      `json:"banner,omitempty"`
	Paths    []AssetPath `json:"paths,omitempty"`
	Vulns    []AssetVuln `json:"vulns,omitempty"`
	Source   string      `json:"source,omitempty"`
}

type AssetPath struct {
	ID         string `json:"id"`
	PortID     string `json:"port_id"`
	Path       string `json:"path"`
	StatusCode int    `json:"status_code"`
	Title      string `json:"title,omitempty"`
	Tech       string `json:"tech,omitempty"`
	Source     string `json:"source,omitempty"`
}

type AssetVuln struct {
	ID       string `json:"id"`
	PortID   string `json:"port_id,omitempty"`
	PathID   string `json:"path_id,omitempty"`
	CVE      string `json:"cve,omitempty"`
	Name     string `json:"name"`
	Severity string `json:"severity"`
	Source   string `json:"source"`
	ExpAvail bool   `json:"exp_available"`
	Detail   string `json:"detail,omitempty"`
}

const (
	ScanTypeSeed           = "seed"
	ScanTypeHostDiscovery  = "host_discovery"
	ScanTypePortScan       = "port_scan"
	ScanTypePathScan       = "path_scan"
	ScanTypeVulnScan       = "vuln_scan"
	ScanTypeExploit        = "exploit"
)

const (
	ScanTaskPending  = "pending"
	ScanTaskRunning  = "running"
	ScanTaskDone     = "done"
	ScanTaskFailed   = "failed"
	ScanTaskSkipped  = "skipped"
)

type ScanTask struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Target    string `json:"target"`
	Status    string `json:"status"`
	ParentID  string `json:"parent_id,omitempty"`
	NodeID    string `json:"node_id,omitempty"`
	Error     string `json:"error,omitempty"`
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

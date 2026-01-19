package types

// ProcessState represents the state of a workflow process instance
type ProcessState string

const (
	StatePending    ProcessState = "PENDING"
	StateInProgress ProcessState = "IN_PROGRESS"
	StateCompleted  ProcessState = "COMPLETED"
	StateFailed     ProcessState = "FAILED"
	StateCancelled  ProcessState = "CANCELLED"
)

// TaskStatus represents the completion status of a task
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "PENDING"
	TaskStatusInProgress TaskStatus = "IN_PROGRESS"
	TaskStatusCompleted  TaskStatus = "COMPLETED"
	TaskStatusFailed     TaskStatus = "FAILED"
)

// ProcessInstance represents a running workflow process instance
type ProcessInstance struct {
	ID            string                 `json:"id"`
	WorkflowID   string                 `json:"workflow_id"`
	State        ProcessState           `json:"state"`
	TaskStatuses map[string]TaskStatus  `json:"task_statuses"`
	Context      map[string]interface{} `json:"context"`
}

// Task represents a workflow task
type Task struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"` // "userTask", "serviceTask", etc.
	Incoming    []string `json:"incoming"` // IDs of incoming sequence flows
	Outgoing    []string `json:"outgoing"` // IDs of outgoing sequence flows
	Required    bool     `json:"required"` // Whether this task is required to proceed
}

// Gateway represents a BPMN gateway (parallel, exclusive, etc.)
type Gateway struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"` // "parallelGateway", "exclusiveGateway"
	Incoming []string `json:"incoming"`
	Outgoing []string `json:"outgoing"`
}

// WorkflowDefinition represents a BPMN workflow definition
type WorkflowDefinition struct {
	ID       string             `json:"id"`
	Name     string             `json:"name"`
	Tasks    map[string]*Task   `json:"tasks"`
	Gateways map[string]*Gateway `json:"gateways"`
	StartID  string             `json:"start_id"`
	EndID    string             `json:"end_id"`
}

// TaskCompletionEvent represents a callback from an OGA SM
type TaskCompletionEvent struct {
	ProcessInstanceID string                 `json:"process_instance_id"`
	TaskID            string                 `json:"task_id"`
	Status            TaskStatus             `json:"status"`
	Data              map[string]interface{} `json:"data,omitempty"`
}


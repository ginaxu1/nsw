package workflow

import (
	"github.com/OpenNSW/nsw/internal/workflow/bpmn"
	"github.com/OpenNSW/nsw/internal/workflow/engine"
	"github.com/OpenNSW/nsw/pkg/types"
)

// WorkflowService provides a high-level interface for workflow operations
type WorkflowService struct {
	executor *engine.Executor
}

// NewWorkflowService creates a new workflow service
func NewWorkflowService() (*WorkflowService, error) {
	exec := engine.NewExecutor()
	
	// Load the mock workflow
	definition, err := bpmn.LoadHypotheticalTradeV1()
	if err != nil {
		return nil, err
	}
	
	if err := exec.RegisterWorkflow(definition); err != nil {
		return nil, err
	}
	
	return &WorkflowService{
		executor: exec,
	}, nil
}

// StartConsignment starts a new consignment workflow process
func (s *WorkflowService) StartConsignment(processInstanceID string, context map[string]interface{}) (*types.ProcessInstance, error) {
	return s.executor.StartProcess("hypothetical_trade_v1", processInstanceID, context)
}

// CompleteTask handles a task completion callback from an OGA SM
func (s *WorkflowService) CompleteTask(event *types.TaskCompletionEvent) error {
	return s.executor.HandleTaskCompletion(event)
}

// GetProcessInstance retrieves a process instance by ID
func (s *WorkflowService) GetProcessInstance(processInstanceID string) (*types.ProcessInstance, error) {
	return s.executor.GetProcessInstance(processInstanceID)
}


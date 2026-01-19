package engine

import (
	"fmt"
	"sync"

	"github.com/OpenNSW/nsw/internal/workflow/state"
	"github.com/OpenNSW/nsw/pkg/types"
)

// Executor executes BPMN workflows with support for parallel gateways
type Executor struct {
	mu           sync.RWMutex
	instances    map[string]*ProcessExecution
	workflows    map[string]*types.WorkflowDefinition
	stateTracker *state.Tracker
}

// ProcessExecution represents an executing process instance
type ProcessExecution struct {
	Instance   *types.ProcessInstance
	Definition *types.WorkflowDefinition
	Tracker    *state.Tracker
}

// NewExecutor creates a new workflow executor
func NewExecutor() *Executor {
	return &Executor{
		instances: make(map[string]*ProcessExecution),
		workflows: make(map[string]*types.WorkflowDefinition),
	}
}

// RegisterWorkflow registers a workflow definition
func (e *Executor) RegisterWorkflow(definition *types.WorkflowDefinition) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if definition.ID == "" {
		return fmt.Errorf("workflow definition must have an ID")
	}
	
	e.workflows[definition.ID] = definition
	return nil
}

// StartProcess starts a new process instance
func (e *Executor) StartProcess(workflowID string, processInstanceID string, context map[string]interface{}) (*types.ProcessInstance, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	definition, exists := e.workflows[workflowID]
	if !exists {
		return nil, fmt.Errorf("workflow %s not found", workflowID)
	}
	
	if _, exists := e.instances[processInstanceID]; exists {
		return nil, fmt.Errorf("process instance %s already exists", processInstanceID)
	}
	
	tracker := state.NewTracker(processInstanceID)
	
	instance := &types.ProcessInstance{
		ID:            processInstanceID,
		WorkflowID:    workflowID,
		State:         types.StateInProgress,
		TaskStatuses:  make(map[string]types.TaskStatus),
		Context:       context,
	}
	
	execution := &ProcessExecution{
		Instance:   instance,
		Definition: definition,
		Tracker:    tracker,
	}
	
	e.instances[processInstanceID] = execution
	
	// Initialize all task statuses
	for taskID := range definition.Tasks {
		instance.TaskStatuses[taskID] = types.TaskStatusPending
		tracker.SetTaskStatus(taskID, types.TaskStatusPending)
	}
	
	// Start from the start event - find the first task after the start event
	// The start event flows to the first task, so we need to find that task
	firstTaskID := e.findFirstTaskAfterStart(definition)
	if firstTaskID == "" {
		return nil, fmt.Errorf("no task found after start event")
	}
	
	if err := e.executeNext(execution, firstTaskID); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}
	
	return instance, nil
}

// HandleTaskCompletion handles a task completion callback from an OGA SM
func (e *Executor) HandleTaskCompletion(event *types.TaskCompletionEvent) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	execution, exists := e.instances[event.ProcessInstanceID]
	if !exists {
		return fmt.Errorf("process instance %s not found", event.ProcessInstanceID)
	}
	
	// Update task status
	execution.Instance.TaskStatuses[event.TaskID] = event.Status
	execution.Tracker.SetTaskStatus(event.TaskID, event.Status)
	
	// Check if this task is part of a parallel group
	definition := execution.Definition
	task, exists := definition.Tasks[event.TaskID]
	if !exists {
		return fmt.Errorf("task %s not found in workflow", event.TaskID)
	}
	
	// If task failed, mark process as failed
	if event.Status == types.TaskStatusFailed {
		execution.Instance.State = types.StateFailed
		return nil
	}
	
	// If task completed, check for parallel gateway join
	if event.Status == types.TaskStatusCompleted {
		// Find outgoing gateway (AND-join)
		for _, outgoingID := range task.Outgoing {
			if gateway, isGateway := definition.Gateways[outgoingID]; isGateway {
				if gateway.Type == "parallelGateway" {
					// Check if all parallel branches are complete
					if e.areAllParallelBranchesComplete(execution, gateway) {
						// Proceed to next task after join
						if err := e.executeNext(execution, gateway.ID); err != nil {
							return fmt.Errorf("failed to proceed after parallel join: %w", err)
						}
					}
				}
			} else {
				// Regular sequence flow to next task
				if err := e.executeNext(execution, outgoingID); err != nil {
					return fmt.Errorf("failed to proceed to next task: %w", err)
				}
			}
		}
	}
	
	return nil
}

// executeNext executes the next task(s) in the workflow
func (e *Executor) executeNext(execution *ProcessExecution, currentElementID string) error {
	definition := execution.Definition
	
	// Check if it's a gateway
	if gateway, isGateway := definition.Gateways[currentElementID]; isGateway {
		if gateway.Type == "parallelGateway" {
			// AND-Split: activate all outgoing tasks in parallel
			return e.activateParallelTasks(execution, gateway)
		}
		// For other gateway types, handle accordingly
		return fmt.Errorf("unsupported gateway type: %s", gateway.Type)
	}
	
	// It's a task
	task, exists := definition.Tasks[currentElementID]
	if !exists {
		return fmt.Errorf("element %s not found", currentElementID)
	}
	
	// Mark task as in progress
	execution.Instance.TaskStatuses[task.ID] = types.TaskStatusInProgress
	execution.Tracker.SetTaskStatus(task.ID, types.TaskStatusInProgress)
	
	return nil
}

// activateParallelTasks activates all tasks after a parallel gateway (AND-Split)
func (e *Executor) activateParallelTasks(execution *ProcessExecution, gateway *types.Gateway) error {
	// Register parallel group
	groupID := fmt.Sprintf("parallel_%s", gateway.ID)
	var taskIDs []string
	
	for _, outgoingID := range gateway.Outgoing {
		if task, isTask := execution.Definition.Tasks[outgoingID]; isTask {
			taskIDs = append(taskIDs, task.ID)
			// Mark tasks as in progress (waiting for OGA callbacks)
			execution.Instance.TaskStatuses[task.ID] = types.TaskStatusInProgress
			execution.Tracker.SetTaskStatus(task.ID, types.TaskStatusInProgress)
		}
	}
	
	if len(taskIDs) > 0 {
		execution.Tracker.RegisterParallelGroup(groupID, taskIDs)
	}
	
	return nil
}

// areAllParallelBranchesComplete checks if all parallel branches leading to a join are complete
func (e *Executor) areAllParallelBranchesComplete(execution *ProcessExecution, joinGateway *types.Gateway) bool {
	// Find all tasks that lead to this join gateway
	var requiredTaskIDs []string
	
	for taskID, task := range execution.Definition.Tasks {
		for _, outgoingID := range task.Outgoing {
			if outgoingID == joinGateway.ID {
				requiredTaskIDs = append(requiredTaskIDs, taskID)
				break
			}
		}
	}
	
	// Check if all required tasks are completed
	for _, taskID := range requiredTaskIDs {
		status := execution.Tracker.GetTaskStatus(taskID)
		if status != types.TaskStatusCompleted && status != types.TaskStatusFailed {
			return false
		}
	}
	
	return true
}

// findFirstTaskAfterStart finds the first task that comes after the start event
func (e *Executor) findFirstTaskAfterStart(definition *types.WorkflowDefinition) string {
	// Find tasks that have the start event in their incoming flows
	for taskID, task := range definition.Tasks {
		for _, incomingID := range task.Incoming {
			if incomingID == definition.StartID {
				return taskID
			}
		}
	}
	
	// If no direct connection, check gateways
	for gatewayID, gateway := range definition.Gateways {
		for _, incomingID := range gateway.Incoming {
			if incomingID == definition.StartID {
				// Gateway is after start, find first task after gateway
				for taskID, task := range definition.Tasks {
					for _, incomingID := range task.Incoming {
						if incomingID == gatewayID {
							return taskID
						}
					}
				}
			}
		}
	}
	
	return ""
}

// GetProcessInstance returns a process instance by ID
func (e *Executor) GetProcessInstance(processInstanceID string) (*types.ProcessInstance, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	execution, exists := e.instances[processInstanceID]
	if !exists {
		return nil, fmt.Errorf("process instance %s not found", processInstanceID)
	}
	
	return execution.Instance, nil
}


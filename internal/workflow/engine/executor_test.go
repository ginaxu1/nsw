package engine

import (
	"testing"

	"github.com/OpenNSW/nsw/pkg/types"
)

func createMockWorkflow() *types.WorkflowDefinition {
	return &types.WorkflowDefinition{
		ID:   "hypothetical_trade_v1",
		Name: "Hypothetical Trade Workflow v1",
		Tasks: map[string]*types.Task{
			"Task_1_Initialization": {
				ID:       "Task_1_Initialization",
				Name:     "Task 1: Initialization",
				Type:     "userTask",
				Incoming: []string{"StartEvent_1"},
				Outgoing: []string{"Gateway_ParallelSplit"},
				Required: true,
			},
			"Task_2_OGA_A": {
				ID:       "Task_2_OGA_A",
				Name:     "Task 2: OGA A Review",
				Type:     "userTask",
				Incoming: []string{"Gateway_ParallelSplit"},
				Outgoing: []string{"Gateway_ParallelJoin"},
				Required: true,
			},
			"Task_3_OGA_B": {
				ID:       "Task_3_OGA_B",
				Name:     "Task 3: OGA B Review",
				Type:     "userTask",
				Incoming: []string{"Gateway_ParallelSplit"},
				Outgoing: []string{"Gateway_ParallelJoin"},
				Required: true,
			},
			"Task_4_Customs": {
				ID:       "Task_4_Customs",
				Name:     "Task 4: Customs Finalization",
				Type:     "userTask",
				Incoming: []string{"Gateway_ParallelJoin"},
				Outgoing: []string{"EndEvent_1"},
				Required: true,
			},
		},
		Gateways: map[string]*types.Gateway{
			"Gateway_ParallelSplit": {
				ID:       "Gateway_ParallelSplit",
				Type:     "parallelGateway",
				Incoming: []string{"Task_1_Initialization"},
				Outgoing: []string{"Task_2_OGA_A", "Task_3_OGA_B"},
			},
			"Gateway_ParallelJoin": {
				ID:       "Gateway_ParallelJoin",
				Type:     "parallelGateway",
				Incoming: []string{"Task_2_OGA_A", "Task_3_OGA_B"},
				Outgoing: []string{"Task_4_Customs"},
			},
		},
		StartID: "StartEvent_1",
		EndID:   "EndEvent_1",
	}
}

func TestExecutor_RegisterWorkflow(t *testing.T) {
	executor := NewExecutor()
	definition := createMockWorkflow()

	err := executor.RegisterWorkflow(definition)
	if err != nil {
		t.Fatalf("failed to register workflow: %v", err)
	}

	// Try to register again - should not error but workflow should exist
	err = executor.RegisterWorkflow(definition)
	if err != nil {
		t.Errorf("re-registering workflow should not error: %v", err)
	}
}

func TestExecutor_RegisterWorkflow_EmptyID(t *testing.T) {
	executor := NewExecutor()
	definition := &types.WorkflowDefinition{
		ID: "", // Empty ID
	}

	err := executor.RegisterWorkflow(definition)
	if err == nil {
		t.Error("expected error when registering workflow with empty ID")
	}
}

func TestExecutor_StartProcess(t *testing.T) {
	executor := NewExecutor()
	definition := createMockWorkflow()

	if err := executor.RegisterWorkflow(definition); err != nil {
		t.Fatalf("failed to register workflow: %v", err)
	}

	processInstanceID := "test-instance-1"
	context := map[string]interface{}{
		"hs_code": "0902.10.00",
		"trader_id": "trader-123",
	}

	instance, err := executor.StartProcess(definition.ID, processInstanceID, context)
	if err != nil {
		t.Fatalf("failed to start process: %v", err)
	}

	if instance.ID != processInstanceID {
		t.Errorf("expected instance ID %s, got %s", processInstanceID, instance.ID)
	}

	if instance.WorkflowID != definition.ID {
		t.Errorf("expected workflow ID %s, got %s", definition.ID, instance.WorkflowID)
	}

	if instance.State != types.StateInProgress {
		t.Errorf("expected state IN_PROGRESS, got %v", instance.State)
	}

	// Verify all tasks are initialized with pending status (except the first task which is activated)
	// Find first task after start event
	firstTaskID := ""
	for taskID, task := range definition.Tasks {
		for _, incomingID := range task.Incoming {
			if incomingID == definition.StartID {
				firstTaskID = taskID
				break
			}
		}
		if firstTaskID != "" {
			break
		}
	}
	
	for taskID := range definition.Tasks {
		expectedStatus := types.TaskStatusPending
		if taskID == firstTaskID {
			expectedStatus = types.TaskStatusInProgress // First task is activated
		}
		if status, exists := instance.TaskStatuses[taskID]; !exists || status != expectedStatus {
			t.Errorf("task %s should be %v, got %v", taskID, expectedStatus, status)
		}
	}
}

func TestExecutor_StartProcess_WorkflowNotFound(t *testing.T) {
	executor := NewExecutor()

	_, err := executor.StartProcess("non-existent-workflow", "test-instance-1", nil)
	if err == nil {
		t.Error("expected error when starting process with non-existent workflow")
	}
}

func TestExecutor_StartProcess_DuplicateInstance(t *testing.T) {
	executor := NewExecutor()
	definition := createMockWorkflow()

	if err := executor.RegisterWorkflow(definition); err != nil {
		t.Fatalf("failed to register workflow: %v", err)
	}

	processInstanceID := "test-instance-1"
	_, err := executor.StartProcess(definition.ID, processInstanceID, nil)
	if err != nil {
		t.Fatalf("failed to start first process: %v", err)
	}

	// Try to start duplicate instance
	_, err = executor.StartProcess(definition.ID, processInstanceID, nil)
	if err == nil {
		t.Error("expected error when starting duplicate process instance")
	}
}

func TestExecutor_HandleTaskCompletion_SequentialFlow(t *testing.T) {
	executor := NewExecutor()
	definition := createMockWorkflow()

	if err := executor.RegisterWorkflow(definition); err != nil {
		t.Fatalf("failed to register workflow: %v", err)
	}

	processInstanceID := "test-instance-1"
	_, err := executor.StartProcess(definition.ID, processInstanceID, nil)
	if err != nil {
		t.Fatalf("failed to start process: %v", err)
	}

	// Complete Task 1 (Initialization)
	event := &types.TaskCompletionEvent{
		ProcessInstanceID: processInstanceID,
		TaskID:            "Task_1_Initialization",
		Status:             types.TaskStatusCompleted,
	}

	err = executor.HandleTaskCompletion(event)
	if err != nil {
		t.Fatalf("failed to handle task completion: %v", err)
	}

	// Verify Task 1 is completed
	updatedInstance, err := executor.GetProcessInstance(processInstanceID)
	if err != nil {
		t.Fatalf("failed to get process instance: %v", err)
	}

	if updatedInstance.TaskStatuses["Task_1_Initialization"] != types.TaskStatusCompleted {
		t.Error("Task_1_Initialization should be COMPLETED")
	}

	// Verify parallel tasks are now in progress (AND-Split activated)
	if updatedInstance.TaskStatuses["Task_2_OGA_A"] != types.TaskStatusInProgress {
		t.Error("Task_2_OGA_A should be IN_PROGRESS after parallel split")
	}
	if updatedInstance.TaskStatuses["Task_3_OGA_B"] != types.TaskStatusInProgress {
		t.Error("Task_3_OGA_B should be IN_PROGRESS after parallel split")
	}
}

func TestExecutor_HandleTaskCompletion_ParallelJoin(t *testing.T) {
	executor := NewExecutor()
	definition := createMockWorkflow()

	if err := executor.RegisterWorkflow(definition); err != nil {
		t.Fatalf("failed to register workflow: %v", err)
	}

	processInstanceID := "test-instance-1"
	_, err := executor.StartProcess(definition.ID, processInstanceID, nil)
	if err != nil {
		t.Fatalf("failed to start process: %v", err)
	}

	// Complete Task 1 to activate parallel split
	event1 := &types.TaskCompletionEvent{
		ProcessInstanceID: processInstanceID,
		TaskID:            "Task_1_Initialization",
		Status:             types.TaskStatusCompleted,
	}
	if err := executor.HandleTaskCompletion(event1); err != nil {
		t.Fatalf("failed to complete Task 1: %v", err)
	}

	// Complete Task 2 (first parallel branch)
	event2 := &types.TaskCompletionEvent{
		ProcessInstanceID: processInstanceID,
		TaskID:            "Task_2_OGA_A",
		Status:             types.TaskStatusCompleted,
	}
	if err := executor.HandleTaskCompletion(event2); err != nil {
		t.Fatalf("failed to complete Task 2: %v", err)
	}

	// Verify Task 4 is not yet activated (waiting for Task 3)
	instance, err := executor.GetProcessInstance(processInstanceID)
	if err != nil {
		t.Fatalf("failed to get process instance: %v", err)
	}

	if instance.TaskStatuses["Task_4_Customs"] != types.TaskStatusPending {
		t.Error("Task_4_Customs should still be PENDING until both parallel tasks complete")
	}

	// Complete Task 3 (second parallel branch)
	event3 := &types.TaskCompletionEvent{
		ProcessInstanceID: processInstanceID,
		TaskID:            "Task_3_OGA_B",
		Status:             types.TaskStatusCompleted,
	}
	if err := executor.HandleTaskCompletion(event3); err != nil {
		t.Fatalf("failed to complete Task 3: %v", err)
	}

	// Verify Task 4 is now activated (AND-Join completed)
	instance, err = executor.GetProcessInstance(processInstanceID)
	if err != nil {
		t.Fatalf("failed to get process instance: %v", err)
	}

	if instance.TaskStatuses["Task_4_Customs"] != types.TaskStatusInProgress {
		t.Error("Task_4_Customs should be IN_PROGRESS after both parallel tasks complete")
	}
}

func TestExecutor_HandleTaskCompletion_TaskFailed(t *testing.T) {
	executor := NewExecutor()
	definition := createMockWorkflow()

	if err := executor.RegisterWorkflow(definition); err != nil {
		t.Fatalf("failed to register workflow: %v", err)
	}

	processInstanceID := "test-instance-1"
	_, err := executor.StartProcess(definition.ID, processInstanceID, nil)
	if err != nil {
		t.Fatalf("failed to start process: %v", err)
	}

	// Fail Task 1
	event := &types.TaskCompletionEvent{
		ProcessInstanceID: processInstanceID,
		TaskID:            "Task_1_Initialization",
		Status:             types.TaskStatusFailed,
	}

	err = executor.HandleTaskCompletion(event)
	if err != nil {
		t.Fatalf("failed to handle task failure: %v", err)
	}

	// Verify process is marked as failed
	instance, err := executor.GetProcessInstance(processInstanceID)
	if err != nil {
		t.Fatalf("failed to get process instance: %v", err)
	}

	if instance.State != types.StateFailed {
		t.Errorf("expected state FAILED, got %v", instance.State)
	}
}

func TestExecutor_HandleTaskCompletion_InvalidInstance(t *testing.T) {
	executor := NewExecutor()

	event := &types.TaskCompletionEvent{
		ProcessInstanceID: "non-existent-instance",
		TaskID:            "Task_1",
		Status:             types.TaskStatusCompleted,
	}

	err := executor.HandleTaskCompletion(event)
	if err == nil {
		t.Error("expected error when handling completion for non-existent instance")
	}
}

func TestExecutor_GetProcessInstance(t *testing.T) {
	executor := NewExecutor()
	definition := createMockWorkflow()

	if err := executor.RegisterWorkflow(definition); err != nil {
		t.Fatalf("failed to register workflow: %v", err)
	}

	processInstanceID := "test-instance-1"
	expectedInstance, err := executor.StartProcess(definition.ID, processInstanceID, nil)
	if err != nil {
		t.Fatalf("failed to start process: %v", err)
	}

	actualInstance, err := executor.GetProcessInstance(processInstanceID)
	if err != nil {
		t.Fatalf("failed to get process instance: %v", err)
	}

	if actualInstance.ID != expectedInstance.ID {
		t.Errorf("expected instance ID %s, got %s", expectedInstance.ID, actualInstance.ID)
	}
}

func TestExecutor_GetProcessInstance_NotFound(t *testing.T) {
	executor := NewExecutor()

	_, err := executor.GetProcessInstance("non-existent-instance")
	if err == nil {
		t.Error("expected error when getting non-existent process instance")
	}
}


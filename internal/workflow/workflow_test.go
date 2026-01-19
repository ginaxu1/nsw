package workflow

import (
	"testing"

	"github.com/OpenNSW/nsw/pkg/types"
)

func TestNewWorkflowService(t *testing.T) {
	service, err := NewWorkflowService()
	if err != nil {
		t.Fatalf("failed to create workflow service: %v", err)
	}

	if service == nil {
		t.Fatal("workflow service should not be nil")
	}
}

func TestWorkflowService_StartConsignment(t *testing.T) {
	service, err := NewWorkflowService()
	if err != nil {
		t.Fatalf("failed to create workflow service: %v", err)
	}

	processInstanceID := "test-consignment-1"
	context := map[string]interface{}{
		"hs_code":  "0902.10.00",
		"trader_id": "trader-123",
	}

	instance, err := service.StartConsignment(processInstanceID, context)
	if err != nil {
		t.Fatalf("failed to start consignment: %v", err)
	}

	if instance.ID != processInstanceID {
		t.Errorf("expected instance ID %s, got %s", processInstanceID, instance.ID)
	}

	if instance.WorkflowID != "hypothetical_trade_v1" {
		t.Errorf("expected workflow ID 'hypothetical_trade_v1', got '%s'", instance.WorkflowID)
	}

	if instance.State != types.StateInProgress {
		t.Errorf("expected state IN_PROGRESS, got %v", instance.State)
	}
}

func TestWorkflowService_CompleteTask(t *testing.T) {
	service, err := NewWorkflowService()
	if err != nil {
		t.Fatalf("failed to create workflow service: %v", err)
	}

	processInstanceID := "test-consignment-1"
	_, err = service.StartConsignment(processInstanceID, nil)
	if err != nil {
		t.Fatalf("failed to start consignment: %v", err)
	}

	event := &types.TaskCompletionEvent{
		ProcessInstanceID: processInstanceID,
		TaskID:            "Task_1_Initialization",
		Status:             types.TaskStatusCompleted,
	}

	err = service.CompleteTask(event)
	if err != nil {
		t.Fatalf("failed to complete task: %v", err)
	}

	// Verify task is completed
	instance, err := service.GetProcessInstance(processInstanceID)
	if err != nil {
		t.Fatalf("failed to get process instance: %v", err)
	}

	if instance.TaskStatuses["Task_1_Initialization"] != types.TaskStatusCompleted {
		t.Error("Task_1_Initialization should be COMPLETED")
	}
}

func TestWorkflowService_CompleteTask_ParallelExecution(t *testing.T) {
	service, err := NewWorkflowService()
	if err != nil {
		t.Fatalf("failed to create workflow service: %v", err)
	}

	processInstanceID := "test-consignment-1"
	_, err = service.StartConsignment(processInstanceID, nil)
	if err != nil {
		t.Fatalf("failed to start consignment: %v", err)
	}

	// Complete Task 1 to activate parallel split
	event1 := &types.TaskCompletionEvent{
		ProcessInstanceID: processInstanceID,
		TaskID:            "Task_1_Initialization",
		Status:             types.TaskStatusCompleted,
	}
	if err := service.CompleteTask(event1); err != nil {
		t.Fatalf("failed to complete Task 1: %v", err)
	}

	// Complete Task 2 (first parallel branch)
	event2 := &types.TaskCompletionEvent{
		ProcessInstanceID: processInstanceID,
		TaskID:            "Task_2_OGA_A",
		Status:             types.TaskStatusCompleted,
	}
	if err := service.CompleteTask(event2); err != nil {
		t.Fatalf("failed to complete Task 2: %v", err)
	}

	// Verify Task 4 is not yet activated
	instance, err := service.GetProcessInstance(processInstanceID)
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
	if err := service.CompleteTask(event3); err != nil {
		t.Fatalf("failed to complete Task 3: %v", err)
	}

	// Verify Task 4 is now activated (AND-Join completed)
	instance, err = service.GetProcessInstance(processInstanceID)
	if err != nil {
		t.Fatalf("failed to get process instance: %v", err)
	}

	if instance.TaskStatuses["Task_4_Customs"] != types.TaskStatusInProgress {
		t.Error("Task_4_Customs should be IN_PROGRESS after both parallel tasks complete")
	}
}

func TestWorkflowService_GetProcessInstance(t *testing.T) {
	service, err := NewWorkflowService()
	if err != nil {
		t.Fatalf("failed to create workflow service: %v", err)
	}

	processInstanceID := "test-consignment-1"
	expectedInstance, err := service.StartConsignment(processInstanceID, nil)
	if err != nil {
		t.Fatalf("failed to start consignment: %v", err)
	}

	actualInstance, err := service.GetProcessInstance(processInstanceID)
	if err != nil {
		t.Fatalf("failed to get process instance: %v", err)
	}

	if actualInstance.ID != expectedInstance.ID {
		t.Errorf("expected instance ID %s, got %s", expectedInstance.ID, actualInstance.ID)
	}
}

func TestWorkflowService_GetProcessInstance_NotFound(t *testing.T) {
	service, err := NewWorkflowService()
	if err != nil {
		t.Fatalf("failed to create workflow service: %v", err)
	}

	_, err = service.GetProcessInstance("non-existent-instance")
	if err == nil {
		t.Error("expected error when getting non-existent process instance")
	}
}


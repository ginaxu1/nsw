package state

import (
	"testing"

	"github.com/OpenNSW/nsw/pkg/types"
)

func TestTracker_SetTaskStatus(t *testing.T) {
	tracker := NewTracker("test-instance-1")

	tests := []struct {
		name     string
		taskID   string
		status   types.TaskStatus
		expected types.TaskStatus
	}{
		{
			name:     "set pending status",
			taskID:   "task-1",
			status:   types.TaskStatusPending,
			expected: types.TaskStatusPending,
		},
		{
			name:     "set in progress status",
			taskID:   "task-1",
			status:   types.TaskStatusInProgress,
			expected: types.TaskStatusInProgress,
		},
		{
			name:     "set completed status",
			taskID:   "task-1",
			status:   types.TaskStatusCompleted,
			expected: types.TaskStatusCompleted,
		},
		{
			name:     "set failed status",
			taskID:   "task-2",
			status:   types.TaskStatusFailed,
			expected: types.TaskStatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker.SetTaskStatus(tt.taskID, tt.status)
			actual := tracker.GetTaskStatus(tt.taskID)
			if actual != tt.expected {
				t.Errorf("expected status %v, got %v", tt.expected, actual)
			}
		})
	}
}

func TestTracker_GetTaskStatus_DefaultPending(t *testing.T) {
	tracker := NewTracker("test-instance-1")

	status := tracker.GetTaskStatus("non-existent-task")
	if status != types.TaskStatusPending {
		t.Errorf("expected default status PENDING, got %v", status)
	}
}

func TestTracker_RegisterParallelGroup(t *testing.T) {
	tracker := NewTracker("test-instance-1")
	groupID := "parallel-group-1"
	taskIDs := []string{"task-1", "task-2", "task-3"}

	tracker.RegisterParallelGroup(groupID, taskIDs)

	// Verify all tasks are registered with pending status
	for _, taskID := range taskIDs {
		status := tracker.GetTaskStatus(taskID)
		if status != types.TaskStatusPending {
			t.Errorf("expected task %s to be PENDING, got %v", taskID, status)
		}
	}
}

func TestTracker_IsParallelGroupComplete(t *testing.T) {
	tests := []struct {
		name           string
		setupStatuses  map[string]types.TaskStatus
		expectedResult bool
	}{
		{
			name: "all tasks completed",
			setupStatuses: map[string]types.TaskStatus{
				"task-1": types.TaskStatusCompleted,
				"task-2": types.TaskStatusCompleted,
			},
			expectedResult: true,
		},
		{
			name: "one task pending",
			setupStatuses: map[string]types.TaskStatus{
				"task-1": types.TaskStatusCompleted,
				"task-2": types.TaskStatusPending,
			},
			expectedResult: false,
		},
		{
			name: "one task in progress",
			setupStatuses: map[string]types.TaskStatus{
				"task-1": types.TaskStatusCompleted,
				"task-2": types.TaskStatusInProgress,
			},
			expectedResult: false,
		},
		{
			name: "one task failed",
			setupStatuses: map[string]types.TaskStatus{
				"task-1": types.TaskStatusCompleted,
				"task-2": types.TaskStatusFailed,
			},
			expectedResult: true, // Failed tasks are considered "complete" for join purposes
		},
		{
			name:           "non-existent group",
			setupStatuses:  map[string]types.TaskStatus{},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := NewTracker("test-instance-1")
			groupID := "parallel-group-1"
			
			var taskIDs []string
			for taskID := range tt.setupStatuses {
				taskIDs = append(taskIDs, taskID)
			}
			
			if len(taskIDs) > 0 {
				tracker.RegisterParallelGroup(groupID, taskIDs)
				// Set statuses after registration
				for taskID, status := range tt.setupStatuses {
					tracker.SetTaskStatus(taskID, status)
				}
			}
			
			result := tracker.IsParallelGroupComplete(groupID)
			if result != tt.expectedResult {
				t.Errorf("expected %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

func TestTracker_GetIncompleteTasksInGroup(t *testing.T) {
	tracker := NewTracker("test-instance-1")
	groupID := "parallel-group-1"
	taskIDs := []string{"task-1", "task-2", "task-3"}

	tracker.RegisterParallelGroup(groupID, taskIDs)
	
	// Mark task-1 as completed
	tracker.SetTaskStatus("task-1", types.TaskStatusCompleted)
	
	incomplete := tracker.GetIncompleteTasksInGroup(groupID)
	
	expectedIncomplete := []string{"task-2", "task-3"}
	if len(incomplete) != len(expectedIncomplete) {
		t.Errorf("expected %d incomplete tasks, got %d", len(expectedIncomplete), len(incomplete))
	}
	
	// Check that task-1 is not in incomplete list
	for _, taskID := range incomplete {
		if taskID == "task-1" {
			t.Errorf("task-1 should not be in incomplete list")
		}
	}
}

func TestTracker_GetAllTaskStatuses(t *testing.T) {
	tracker := NewTracker("test-instance-1")
	
	tracker.SetTaskStatus("task-1", types.TaskStatusInProgress)
	tracker.SetTaskStatus("task-2", types.TaskStatusCompleted)
	tracker.SetTaskStatus("task-3", types.TaskStatusPending)
	
	allStatuses := tracker.GetAllTaskStatuses()
	
	if len(allStatuses) != 3 {
		t.Errorf("expected 3 task statuses, got %d", len(allStatuses))
	}
	
	if allStatuses["task-1"] != types.TaskStatusInProgress {
		t.Errorf("task-1 should be IN_PROGRESS")
	}
	if allStatuses["task-2"] != types.TaskStatusCompleted {
		t.Errorf("task-2 should be COMPLETED")
	}
	if allStatuses["task-3"] != types.TaskStatusPending {
		t.Errorf("task-3 should be PENDING")
	}
}


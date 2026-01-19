package state

import (
	"sync"

	"github.com/OpenNSW/nsw/pkg/types"
)

// Tracker manages the state of parallel branches in a workflow instance
type Tracker struct {
	mu                sync.RWMutex
	processInstanceID string
	taskStatuses     map[string]types.TaskStatus
	parallelGroups   map[string]*ParallelGroup
}

// ParallelGroup tracks a group of parallel tasks that must all complete
type ParallelGroup struct {
	ID       string
	TaskIDs  []string
	Required map[string]bool // tracks which tasks are required
	mu       sync.RWMutex
}

// NewTracker creates a new state tracker for a process instance
func NewTracker(processInstanceID string) *Tracker {
	return &Tracker{
		processInstanceID: processInstanceID,
		taskStatuses:      make(map[string]types.TaskStatus),
		parallelGroups:    make(map[string]*ParallelGroup),
	}
}

// SetTaskStatus updates the status of a task
func (t *Tracker) SetTaskStatus(taskID string, status types.TaskStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.taskStatuses[taskID] = status
}

// GetTaskStatus returns the current status of a task
func (t *Tracker) GetTaskStatus(taskID string) types.TaskStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if status, exists := t.taskStatuses[taskID]; exists {
		return status
	}
	return types.TaskStatusPending
}

// RegisterParallelGroup registers a group of tasks that must execute in parallel
func (t *Tracker) RegisterParallelGroup(groupID string, taskIDs []string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	required := make(map[string]bool)
	for _, taskID := range taskIDs {
		required[taskID] = true
		t.taskStatuses[taskID] = types.TaskStatusPending
	}
	
	t.parallelGroups[groupID] = &ParallelGroup{
		ID:       groupID,
		TaskIDs:  taskIDs,
		Required: required,
	}
}

// IsParallelGroupComplete checks if all tasks in a parallel group are completed
func (t *Tracker) IsParallelGroupComplete(groupID string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	group, exists := t.parallelGroups[groupID]
	if !exists {
		return false
	}
	
	group.mu.RLock()
	defer group.mu.RUnlock()
	
	for _, taskID := range group.TaskIDs {
		status := t.taskStatuses[taskID]
		if status != types.TaskStatusCompleted && status != types.TaskStatusFailed {
			return false
		}
	}
	return true
}

// GetIncompleteTasksInGroup returns task IDs that are not yet completed in a parallel group
func (t *Tracker) GetIncompleteTasksInGroup(groupID string) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	group, exists := t.parallelGroups[groupID]
	if !exists {
		return nil
	}
	
	group.mu.RLock()
	defer group.mu.RUnlock()
	
	var incomplete []string
	for _, taskID := range group.TaskIDs {
		status := t.taskStatuses[taskID]
		if status != types.TaskStatusCompleted && status != types.TaskStatusFailed {
			incomplete = append(incomplete, taskID)
		}
	}
	return incomplete
}

// GetAllTaskStatuses returns a copy of all task statuses
func (t *Tracker) GetAllTaskStatuses() map[string]types.TaskStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	result := make(map[string]types.TaskStatus)
	for k, v := range t.taskStatuses {
		result[k] = v
	}
	return result
}


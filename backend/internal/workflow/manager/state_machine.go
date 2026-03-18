package manager

import (
	"context"
	"fmt"
	"sort"

	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/internal/workflow/model"
)

// StateTransitionResult represents the result of a workflow node state transition.
type StateTransitionResult struct {
	UpdatedNodes     []model.WorkflowNode
	NewReadyNodes    []model.WorkflowNode
	WorkflowFinished bool
}

// WorkflowCompletionConfig holds configuration for determining workflow completion.
type WorkflowCompletionConfig struct {
	EndNodeID *string
}

// WorkflowNodeStateMachine handles workflow node state transitions and dependency propagation.
type WorkflowNodeStateMachine struct {
	nodeRepo WorkflowNodeRepository
}

// NewWorkflowNodeStateMachine creates a new instance of WorkflowNodeStateMachine.
func NewWorkflowNodeStateMachine(nodeRepo WorkflowNodeRepository) *WorkflowNodeStateMachine {
	return &WorkflowNodeStateMachine{nodeRepo: nodeRepo}
}

// TransitionToCompleted transitions a workflow node to COMPLETED state and propagates updates.
func (sm *WorkflowNodeStateMachine) TransitionToCompleted(
	ctx context.Context,
	tx *gorm.DB,
	node *model.WorkflowNode,
	updateReq *model.UpdateWorkflowNodeDTO,
	completionConfig ...*WorkflowCompletionConfig,
) (*StateTransitionResult, error) {
	if node == nil {
		return nil, fmt.Errorf("node cannot be nil")
	}

	if node.State == model.WorkflowNodeStateCompleted {
		return &StateTransitionResult{UpdatedNodes: []model.WorkflowNode{}, NewReadyNodes: []model.WorkflowNode{}, WorkflowFinished: false}, nil
	}

	if !sm.canTransitionToCompleted(node.State) {
		return nil, fmt.Errorf("cannot transition node %s from state %s to COMPLETED", node.ID, node.State)
	}

	node.State = model.WorkflowNodeStateCompleted
	node.ExtendedState = updateReq.ExtendedState
	node.Outcome = updateReq.Outcome
	nodesToUpdate := []model.WorkflowNode{*node}
	updatedNodeIndex := map[string]int{node.ID: 0}

	allNodes, err := sm.getSiblingNodes(ctx, tx, node)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve sibling workflow nodes: %w", err)
	}

	var cfg *WorkflowCompletionConfig
	if len(completionConfig) > 0 {
		cfg = completionConfig[0]
	}

	nodeStateMap := sm.buildNodeStateMap(allNodes)
	nodeStateMap[node.ID] = *node

	unlockedNodes := sm.unlockDependentNodes(allNodes, nodeStateMap)
	for _, unlockedNode := range unlockedNodes {
		if _, exists := updatedNodeIndex[unlockedNode.ID]; !exists {
			nodesToUpdate = append(nodesToUpdate, unlockedNode)
			updatedNodeIndex[unlockedNode.ID] = len(nodesToUpdate) - 1
		}
	}

	if cfg != nil && cfg.EndNodeID != nil {
		endNodeID := *cfg.EndNodeID
		if endNode, exists := nodeStateMap[endNodeID]; exists && endNode.State == model.WorkflowNodeStateReady {
			endNode.State = model.WorkflowNodeStateCompleted
			nodeStateMap[endNodeID] = endNode
			if index, found := updatedNodeIndex[endNodeID]; found {
				nodesToUpdate[index] = endNode
			} else {
				nodesToUpdate = append(nodesToUpdate, endNode)
				updatedNodeIndex[endNodeID] = len(nodesToUpdate) - 1
			}
		}
	}

	newReadyNodes := make([]model.WorkflowNode, 0, len(unlockedNodes))
	for _, unlockedNode := range unlockedNodes {
		if nodeStateMap[unlockedNode.ID].State == model.WorkflowNodeStateReady {
			newReadyNodes = append(newReadyNodes, unlockedNode)
		}
	}

	sm.sortNodesByID(nodesToUpdate)
	allCompleted := sm.evaluateWorkflowCompletion(allNodes, nodeStateMap, cfg)

	if err := sm.nodeRepo.UpdateWorkflowNodesInTx(ctx, tx, nodesToUpdate); err != nil {
		return nil, fmt.Errorf("failed to update workflow nodes: %w", err)
	}

	return &StateTransitionResult{UpdatedNodes: nodesToUpdate, NewReadyNodes: newReadyNodes, WorkflowFinished: allCompleted}, nil
}

// TransitionToFailed transitions a workflow node to FAILED state.
func (sm *WorkflowNodeStateMachine) TransitionToFailed(
	ctx context.Context,
	tx *gorm.DB,
	node *model.WorkflowNode,
	updateReq *model.UpdateWorkflowNodeDTO,
) error {
	if node == nil {
		return fmt.Errorf("node cannot be nil")
	}

	if node.State == model.WorkflowNodeStateFailed {
		return nil
	}

	if !sm.canTransitionToFailed(node.State) {
		return fmt.Errorf("cannot transition node %s from state %s to FAILED", node.ID, node.State)
	}

	node.State = model.WorkflowNodeStateFailed
	node.ExtendedState = updateReq.ExtendedState
	if err := sm.nodeRepo.UpdateWorkflowNodesInTx(ctx, tx, []model.WorkflowNode{*node}); err != nil {
		return fmt.Errorf("failed to update workflow node %s to FAILED state: %w", node.ID, err)
	}

	return nil
}

// TransitionToInProgress transitions a workflow node to IN_PROGRESS state.
func (sm *WorkflowNodeStateMachine) TransitionToInProgress(
	ctx context.Context,
	tx *gorm.DB,
	node *model.WorkflowNode,
	updateReq *model.UpdateWorkflowNodeDTO,
) error {
	if node == nil {
		return fmt.Errorf("node cannot be nil")
	}

	if updateReq.ExtendedState == node.ExtendedState && node.State == model.WorkflowNodeStateInProgress {
		return nil
	} else if !sm.canTransitionToInProgress(node.State) {
		return fmt.Errorf("cannot transition node %s from state %s to IN_PROGRESS", node.ID, node.State)
	} else {
		node.State = model.WorkflowNodeStateInProgress
	}
	node.ExtendedState = updateReq.ExtendedState
	if err := sm.nodeRepo.UpdateWorkflowNodesInTx(ctx, tx, []model.WorkflowNode{*node}); err != nil {
		return fmt.Errorf("failed to update workflow node %s to IN_PROGRESS state: %w", node.ID, err)
	}

	return nil
}

// InitializeNodesFromTemplates creates workflow nodes from templates and sets up their dependencies.
func (sm *WorkflowNodeStateMachine) InitializeNodesFromTemplates(
	ctx context.Context,
	tx *gorm.DB,
	workflowID string,
	nodeTemplates []model.WorkflowNodeTemplate,
) ([]model.WorkflowNode, []model.WorkflowNode, *string, error) {
	if len(nodeTemplates) == 0 {
		return []model.WorkflowNode{}, []model.WorkflowNode{}, nil, nil
	}

	templateMap := make(map[string]model.WorkflowNodeTemplate)
	for _, t := range nodeTemplates {
		templateMap[t.ID] = t
	}

	workflowNodes := make([]model.WorkflowNode, 0, len(nodeTemplates))
	for _, template := range nodeTemplates {
		workflowNode := model.WorkflowNode{
			WorkflowID:             workflowID,
			WorkflowNodeTemplateID: template.ID,
			State:                  model.WorkflowNodeStateLocked,
			DependsOn:              model.StringArray(make([]string, 0)),
		}
		workflowNodes = append(workflowNodes, workflowNode)
	}

	createdNodes, err := sm.nodeRepo.CreateWorkflowNodesInTx(ctx, tx, workflowNodes)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create workflow nodes: %w", err)
	}

	nodeByTemplateID := make(map[string]model.WorkflowNode)
	for _, node := range createdNodes {
		nodeByTemplateID[node.WorkflowNodeTemplateID] = node
	}

	var nodesToUpdate []model.WorkflowNode
	var newReadyNodes []model.WorkflowNode
	templateToNodeID := make(map[string]string)
	for templateID, node := range nodeByTemplateID {
		templateToNodeID[templateID] = node.ID
	}

	var endNodeID *string
	for i, node := range createdNodes {
		template, exists := templateMap[node.WorkflowNodeTemplateID]
		if !exists {
			return nil, nil, nil, fmt.Errorf("workflow node template with ID %s not found", node.WorkflowNodeTemplateID)
		}

		if template.Type == model.WorkFlowNodeTypeEndNode {
			endNodeID = &createdNodes[i].ID
		}

		dependsOnNodeIDs := make([]string, 0)
		for _, dependsOnTemplateID := range template.DependsOn {
			if depNode, found := nodeByTemplateID[dependsOnTemplateID]; found {
				dependsOnNodeIDs = append(dependsOnNodeIDs, depNode.ID)
			}
		}
		createdNodes[i].DependsOn = dependsOnNodeIDs

		if template.UnlockConfiguration != nil {
			resolvedConfig, err := template.UnlockConfiguration.ResolveToInstanceIDs(templateToNodeID)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to resolve unlock configuration for node template %s: %w", template.ID, err)
			}
			createdNodes[i].UnlockConfiguration = resolvedConfig
		}

		needsUpdate := false
		if len(dependsOnNodeIDs) > 0 {
			needsUpdate = true
		}
		if createdNodes[i].UnlockConfiguration != nil {
			needsUpdate = true
		}
		if len(dependsOnNodeIDs) == 0 && createdNodes[i].UnlockConfiguration == nil {
			createdNodes[i].State = model.WorkflowNodeStateReady
			newReadyNodes = append(newReadyNodes, createdNodes[i])
			needsUpdate = true
		}

		if needsUpdate {
			nodesToUpdate = append(nodesToUpdate, createdNodes[i])
		}
	}

	if len(nodesToUpdate) > 0 {
		if err := sm.nodeRepo.UpdateWorkflowNodesInTx(ctx, tx, nodesToUpdate); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to update workflow nodes with dependencies: %w", err)
		}
	}

	return createdNodes, newReadyNodes, endNodeID, nil
}

func (sm *WorkflowNodeStateMachine) unlockDependentNodes(
	allNodes []model.WorkflowNode,
	nodeStateMap map[string]model.WorkflowNode,
) []model.WorkflowNode {
	var unlockedNodes []model.WorkflowNode
	for _, node := range allNodes {
		if node.State != model.WorkflowNodeStateLocked {
			continue
		}

		if sm.areDependenciesMet(node, nodeStateMap) {
			node.State = model.WorkflowNodeStateReady
			unlockedNodes = append(unlockedNodes, node)
			nodeStateMap[node.ID] = node
		}
	}

	return unlockedNodes
}

func (sm *WorkflowNodeStateMachine) areDependenciesMet(
	node model.WorkflowNode,
	nodeMap map[string]model.WorkflowNode,
) bool {
	if node.UnlockConfiguration != nil {
		return node.UnlockConfiguration.Evaluate(nodeMap)
	}

	for _, depID := range node.DependsOn {
		depNode, exists := nodeMap[depID]
		if !exists {
			return false
		}
		if depNode.State != model.WorkflowNodeStateCompleted {
			return false
		}
	}
	return true
}

func (sm *WorkflowNodeStateMachine) evaluateWorkflowCompletion(
	allNodes []model.WorkflowNode,
	nodeStateMap map[string]model.WorkflowNode,
	config *WorkflowCompletionConfig,
) bool {
	if config != nil && config.EndNodeID != nil {
		endNodeID := *config.EndNodeID
		endNode, exists := nodeStateMap[endNodeID]
		if !exists {
			return false
		}
		return endNode.State == model.WorkflowNodeStateCompleted
	}

	for _, node := range allNodes {
		state := node.State
		if current, exists := nodeStateMap[node.ID]; exists {
			state = current.State
		}
		if state != model.WorkflowNodeStateCompleted {
			return false
		}
	}

	return true
}

func (sm *WorkflowNodeStateMachine) buildNodeStateMap(allNodes []model.WorkflowNode) map[string]model.WorkflowNode {
	nodeStateMap := make(map[string]model.WorkflowNode, len(allNodes))
	for _, node := range allNodes {
		nodeStateMap[node.ID] = node
	}
	return nodeStateMap
}

func (sm *WorkflowNodeStateMachine) canTransitionToCompleted(currentState model.WorkflowNodeState) bool {
	return currentState == model.WorkflowNodeStateReady || currentState == model.WorkflowNodeStateInProgress
}

func (sm *WorkflowNodeStateMachine) canTransitionToFailed(currentState model.WorkflowNodeState) bool {
	return currentState == model.WorkflowNodeStateReady || currentState == model.WorkflowNodeStateInProgress
}

func (sm *WorkflowNodeStateMachine) canTransitionToInProgress(currentState model.WorkflowNodeState) bool {
	return currentState == model.WorkflowNodeStateReady || currentState == model.WorkflowNodeStateFailed || currentState == model.WorkflowNodeStateInProgress
}

func (sm *WorkflowNodeStateMachine) sortNodesByID(nodes []model.WorkflowNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})
}

func (sm *WorkflowNodeStateMachine) getSiblingNodes(ctx context.Context, tx *gorm.DB, node *model.WorkflowNode) ([]model.WorkflowNode, error) {
	return sm.nodeRepo.GetWorkflowNodesByWorkflowIDInTx(ctx, tx, node.WorkflowID)
}

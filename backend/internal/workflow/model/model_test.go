package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestWorkflowTemplate_GetNodeTemplateIDs(t *testing.T) {
	nodeID1 := uuid.New()
	nodeID2 := uuid.New()
	wt := WorkflowTemplate{
		NodeTemplates: UUIDArray{nodeID1, nodeID2},
	}

	ids := wt.GetNodeTemplateIDs()
	assert.Len(t, ids, 2)
	assert.Contains(t, ids, nodeID1)
	assert.Contains(t, ids, nodeID2)
}

package bpmn

import (
	"embed"

	"github.com/OpenNSW/nsw/pkg/types"
)

//go:embed hypothetical_trade_v1.bpmn
var bpmnFiles embed.FS

// LoadWorkflow loads a workflow definition from an embedded BPMN file
func LoadWorkflow(filename string) (*types.WorkflowDefinition, error) {
	file, err := bpmnFiles.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	return Parse(file)
}

// LoadHypotheticalTradeV1 loads the mock hypothetical trade workflow
func LoadHypotheticalTradeV1() (*types.WorkflowDefinition, error) {
	return LoadWorkflow("hypothetical_trade_v1.bpmn")
}


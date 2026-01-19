package bpmn

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/OpenNSW/nsw/pkg/types"
)

// BPMN2 represents the root of a BPMN 2.0 XML document
type BPMN2 struct {
	XMLName xml.Name `xml:"definitions"`
	Process Process  `xml:"process"`
}

// Process represents a BPMN process
type Process struct {
	XMLName     xml.Name     `xml:"process"`
	ID          string       `xml:"id,attr"`
	Name        string       `xml:"name,attr"`
	StartEvents []StartEvent `xml:"startEvent"`
	Tasks       []Task       `xml:"userTask"`
	Gateways    []Gateway    `xml:"parallelGateway"`
	EndEvents   []EndEvent   `xml:"endEvent"`
	Flows       []Flow       `xml:"sequenceFlow"`
}

// StartEvent represents a BPMN start event
type StartEvent struct {
	XMLName xml.Name `xml:"startEvent"`
	ID      string   `xml:"id,attr"`
	Name    string   `xml:"name,attr"`
}

// Task represents a BPMN user task
type Task struct {
	XMLName xml.Name `xml:"userTask"`
	ID      string   `xml:"id,attr"`
	Name    string   `xml:"name,attr"`
}

// Gateway represents a BPMN parallel gateway
type Gateway struct {
	XMLName xml.Name `xml:"parallelGateway"`
	ID      string   `xml:"id,attr"`
	Name    string   `xml:"name,attr"`
}

// EndEvent represents a BPMN end event
type EndEvent struct {
	XMLName xml.Name `xml:"endEvent"`
	ID      string   `xml:"id,attr"`
	Name    string   `xml:"name,attr"`
}

// Flow represents a BPMN sequence flow
type Flow struct {
	XMLName    xml.Name `xml:"sequenceFlow"`
	ID         string   `xml:"id,attr"`
	SourceRef  string   `xml:"sourceRef,attr"`
	TargetRef  string   `xml:"targetRef,attr"`
}

// Parse parses a BPMN 2.0 XML file and converts it to a WorkflowDefinition
func Parse(r io.Reader) (*types.WorkflowDefinition, error) {
	var bpmn BPMN2
	decoder := xml.NewDecoder(r)
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity
	
	if err := decoder.Decode(&bpmn); err != nil {
		// Try parsing without namespace prefix as fallback
		var bpmnAlt struct {
			XMLName xml.Name `xml:"definitions"`
			Process Process  `xml:"process"`
		}
		decoder = xml.NewDecoder(r)
		decoder.Strict = false
		if err2 := decoder.Decode(&bpmnAlt); err2 != nil {
			return nil, fmt.Errorf("failed to parse BPMN XML: %w (fallback also failed: %v)", err, err2)
		}
		bpmn = BPMN2{
			XMLName: bpmnAlt.XMLName,
			Process: bpmnAlt.Process,
		}
	}
	
	definition := &types.WorkflowDefinition{
		ID:       bpmn.Process.ID,
		Name:     bpmn.Process.Name,
		Tasks:    make(map[string]*types.Task),
		Gateways: make(map[string]*types.Gateway),
	}
	
	// Build flow map for quick lookup
	flowMap := make(map[string][]string) // source -> targets
	for _, flow := range bpmn.Process.Flows {
		flowMap[flow.SourceRef] = append(flowMap[flow.SourceRef], flow.TargetRef)
	}
	
	reverseFlowMap := make(map[string][]string) // target -> sources
	for _, flow := range bpmn.Process.Flows {
		reverseFlowMap[flow.TargetRef] = append(reverseFlowMap[flow.TargetRef], flow.SourceRef)
	}
	
	// Parse start event
	if len(bpmn.Process.StartEvents) == 0 {
		return nil, fmt.Errorf("workflow must have at least one start event")
	}
	startEvent := bpmn.Process.StartEvents[0]
	definition.StartID = startEvent.ID
	
	// Parse tasks
	for _, task := range bpmn.Process.Tasks {
		definition.Tasks[task.ID] = &types.Task{
			ID:       task.ID,
			Name:     task.Name,
			Type:     "userTask",
			Incoming: reverseFlowMap[task.ID],
			Outgoing: flowMap[task.ID],
			Required: true,
		}
	}
	
	// Parse gateways
	for _, gateway := range bpmn.Process.Gateways {
		definition.Gateways[gateway.ID] = &types.Gateway{
			ID:       gateway.ID,
			Type:     "parallelGateway",
			Incoming: reverseFlowMap[gateway.ID],
			Outgoing: flowMap[gateway.ID],
		}
	}
	
	// Parse end event
	if len(bpmn.Process.EndEvents) > 0 {
		endEvent := bpmn.Process.EndEvents[0]
		definition.EndID = endEvent.ID
	}
	
	return definition, nil
}


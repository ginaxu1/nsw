package bpmn

import (
	"strings"
	"testing"
)

const mockBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn2:definitions xmlns:bpmn2="http://www.omg.org/spec/BPMN/20100524/MODEL"
                   id="test-definitions"
                   targetNamespace="http://nsw.gov.lk/trade">
  <bpmn2:process id="test_workflow" name="Test Workflow" isExecutable="true">
    <bpmn2:startEvent id="StartEvent_1" name="Start"/>
    <bpmn2:userTask id="Task_1" name="Task 1"/>
    <bpmn2:parallelGateway id="Gateway_Split" name="Split"/>
    <bpmn2:userTask id="Task_2" name="Task 2"/>
    <bpmn2:userTask id="Task_3" name="Task 3"/>
    <bpmn2:parallelGateway id="Gateway_Join" name="Join"/>
    <bpmn2:userTask id="Task_4" name="Task 4"/>
    <bpmn2:endEvent id="EndEvent_1" name="End"/>
    <bpmn2:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="Task_1"/>
    <bpmn2:sequenceFlow id="Flow_2" sourceRef="Task_1" targetRef="Gateway_Split"/>
    <bpmn2:sequenceFlow id="Flow_3" sourceRef="Gateway_Split" targetRef="Task_2"/>
    <bpmn2:sequenceFlow id="Flow_4" sourceRef="Gateway_Split" targetRef="Task_3"/>
    <bpmn2:sequenceFlow id="Flow_5" sourceRef="Task_2" targetRef="Gateway_Join"/>
    <bpmn2:sequenceFlow id="Flow_6" sourceRef="Task_3" targetRef="Gateway_Join"/>
    <bpmn2:sequenceFlow id="Flow_7" sourceRef="Gateway_Join" targetRef="Task_4"/>
    <bpmn2:sequenceFlow id="Flow_8" sourceRef="Task_4" targetRef="EndEvent_1"/>
  </bpmn2:process>
</bpmn2:definitions>`

func TestParse(t *testing.T) {
	reader := strings.NewReader(mockBPMN)
	definition, err := Parse(reader)
	if err != nil {
		t.Fatalf("failed to parse BPMN: %v", err)
	}

	if definition.ID != "test_workflow" {
		t.Errorf("expected workflow ID 'test_workflow', got '%s'", definition.ID)
	}

	if definition.Name != "Test Workflow" {
		t.Errorf("expected workflow name 'Test Workflow', got '%s'", definition.Name)
	}
}

func TestParse_Tasks(t *testing.T) {
	reader := strings.NewReader(mockBPMN)
	definition, err := Parse(reader)
	if err != nil {
		t.Fatalf("failed to parse BPMN: %v", err)
	}

	expectedTasks := []string{"Task_1", "Task_2", "Task_3", "Task_4"}
	if len(definition.Tasks) != len(expectedTasks) {
		t.Errorf("expected %d tasks, got %d", len(expectedTasks), len(definition.Tasks))
	}

	for _, taskID := range expectedTasks {
		task, exists := definition.Tasks[taskID]
		if !exists {
			t.Errorf("task %s not found", taskID)
			continue
		}

		if task.ID != taskID {
			t.Errorf("task ID mismatch: expected %s, got %s", taskID, task.ID)
		}

		if task.Type != "userTask" {
			t.Errorf("expected task type 'userTask', got '%s'", task.Type)
		}
	}
}

func TestParse_Gateways(t *testing.T) {
	reader := strings.NewReader(mockBPMN)
	definition, err := Parse(reader)
	if err != nil {
		t.Fatalf("failed to parse BPMN: %v", err)
	}

	expectedGateways := []string{"Gateway_Split", "Gateway_Join"}
	if len(definition.Gateways) != len(expectedGateways) {
		t.Errorf("expected %d gateways, got %d", len(expectedGateways), len(definition.Gateways))
	}

	for _, gatewayID := range expectedGateways {
		gateway, exists := definition.Gateways[gatewayID]
		if !exists {
			t.Errorf("gateway %s not found", gatewayID)
			continue
		}

		if gateway.ID != gatewayID {
			t.Errorf("gateway ID mismatch: expected %s, got %s", gatewayID, gateway.ID)
		}

		if gateway.Type != "parallelGateway" {
			t.Errorf("expected gateway type 'parallelGateway', got '%s'", gateway.Type)
		}
	}
}

func TestParse_SequenceFlows(t *testing.T) {
	reader := strings.NewReader(mockBPMN)
	definition, err := Parse(reader)
	if err != nil {
		t.Fatalf("failed to parse BPMN: %v", err)
	}

	// Check Task_1 outgoing flows
	task1 := definition.Tasks["Task_1"]
	if len(task1.Outgoing) != 1 {
		t.Errorf("Task_1 should have 1 outgoing flow, got %d", len(task1.Outgoing))
	}
	if task1.Outgoing[0] != "Gateway_Split" {
		t.Errorf("Task_1 should flow to Gateway_Split, got %s", task1.Outgoing[0])
	}

	// Check Gateway_Split outgoing flows (should have 2)
	splitGateway := definition.Gateways["Gateway_Split"]
	if len(splitGateway.Outgoing) != 2 {
		t.Errorf("Gateway_Split should have 2 outgoing flows, got %d", len(splitGateway.Outgoing))
	}

	// Check Gateway_Join incoming flows (should have 2)
	joinGateway := definition.Gateways["Gateway_Join"]
	if len(joinGateway.Incoming) != 2 {
		t.Errorf("Gateway_Join should have 2 incoming flows, got %d", len(joinGateway.Incoming))
	}
}

func TestParse_StartEvent(t *testing.T) {
	reader := strings.NewReader(mockBPMN)
	definition, err := Parse(reader)
	if err != nil {
		t.Fatalf("failed to parse BPMN: %v", err)
	}

	if definition.StartID != "StartEvent_1" {
		t.Errorf("expected start ID 'StartEvent_1', got '%s'", definition.StartID)
	}
}

func TestParse_EndEvent(t *testing.T) {
	reader := strings.NewReader(mockBPMN)
	definition, err := Parse(reader)
	if err != nil {
		t.Fatalf("failed to parse BPMN: %v", err)
	}

	if definition.EndID != "EndEvent_1" {
		t.Errorf("expected end ID 'EndEvent_1', got '%s'", definition.EndID)
	}
}

func TestParse_NoStartEvent(t *testing.T) {
	invalidBPMN := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn2:definitions xmlns:bpmn2="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn2:process id="test_workflow" name="Test Workflow" isExecutable="true">
  </bpmn2:process>
</bpmn2:definitions>`

	reader := strings.NewReader(invalidBPMN)
	_, err := Parse(reader)
	if err == nil {
		t.Error("expected error when parsing BPMN without start event")
	}
}

func TestParse_InvalidXML(t *testing.T) {
	invalidXML := "not valid xml"
	reader := strings.NewReader(invalidXML)
	
	_, err := Parse(reader)
	if err == nil {
		t.Error("expected error when parsing invalid XML")
	}
}


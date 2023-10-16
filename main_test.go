package main

import (
	"message-bot-demo/flows"
	"testing"
)

// TODO include more tests for buttons, images etc.
func TestTemplateSubstitution(t *testing.T) {
	flow := flows.DemoFlowFactory()
	requestFlowData := flows.DemoRequestFlowData()

    expected := "Welcome to the demo flow John! Are you interested in our coupon promotion?"
	populated, _ := flows.PopulateNodeWithParams(flow.Nodes[0], requestFlowData.Params)
	actual := populated.Body

	if expected != actual {
		t.Errorf("Expected %q but got %q", expected, actual)
	}
}

// TODO add tests for flow state transitions

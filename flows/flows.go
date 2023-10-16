package flows

import (
	"errors"
	"fmt"
	"strings"
	"text/template"
)

// params for substitution in template strings of the flow
type RequestMessageNodeParams map[string]string

// track visits for each state of a flow, keys are state id's and
// values number of visits
type FlowStats struct {
	Stats map[int]int
}

// track active flow, active state and user specific params for
// template substitution
type FlowState struct {
	Flow   string                   `json:"flow,omitempty"`
	State  int                      `json:"state,omitempty"`
	Params RequestMessageNodeParams `json:"params,omitempty"`
}

// request statistics for a specific flow
type RequestStats struct {
	Flow string `json:"flow,omitempty"`
}

// each flow comprises of MessageNodes, which can have a body text,
// buttons with text, an image and transitions to other nodes in the
// flow. Int's are used for node ids
// TODO introduce a new type for node ids
type MessageNode struct {
	NodeId    int            `json:"node_id,omitempty"`
	Body      string         `json:"body,omitempty"`
	Button    []string       `json:"button,omitempty"`
	Image     string         `json:"image,omitempty"`
	NextNodes map[string]int `json:"next_nodes,omitempty"`
}

// functional options for constructing MessageNodes
type MessageNodeOption func(messageNode *MessageNode)

type Flow struct {
	Name       string               `json:"name,omitempty"`
	NextNodeId int                  `json:"next_node_id,omitempty"`
	Nodes      map[int]*MessageNode `json:"nodes,omitempty"`
}

// data for initiating a flow-based conversation using a specific
// flow and state, along with parameters for template substitution
type RequestSendData struct {
	UserId  string                   `json:"user_id,omitempty"`
	Flow    string                   `json:"flow,omitempty"`
	StateId int                      `json:"state_id,omitempty"`
	Params  RequestMessageNodeParams `json:"params,omitempty"`
}

func NewMessageNode(body string, opts ...MessageNodeOption) *MessageNode {
	messageNode := &MessageNode{Body: body}
	messageNode.NextNodes = map[string]int{}

	for _, opt := range opts {
		opt(messageNode)
	}

	return messageNode
}

func WithButtons(buttons ...string) MessageNodeOption {
	return func(messageNode *MessageNode) {
		messageNode.Button = buttons
	}
}

func WithImage(image string) MessageNodeOption {
	return func(messageNode *MessageNode) {
		messageNode.Image = image
	}
}

func NewFlow(flowName string) *Flow {
	flow := &Flow{}
	flow.Nodes = map[int]*MessageNode{}
	flow.Name = flowName
	return flow
}

func AddNode(flow *Flow, messageNode *MessageNode) int {
	currentId := flow.NextNodeId
	messageNode.NodeId = currentId
	flow.Nodes[currentId] = messageNode
	flow.NextNodeId++
	return currentId
}

func AddTransition(flow *Flow, startNode, endNode int, message string) error {
	start, okStart := flow.Nodes[startNode]
	_, okEnd := flow.Nodes[endNode]

	if okStart && okEnd {
		start.NextNodes[message] = endNode
		start.Button = append(start.Button, message)
		return nil
	}

	return errors.New("flow does not contain both nodes")
}

// helper function for stdin/stdout interaction with the chat bot
func SendToStdout(messageNode *MessageNode) {
	fmt.Println(messageNode.ToString())
}

// for testing purposes
// TODO delete this
func DemoRequestFlowData() RequestSendData {
	r := RequestSendData{
		"123456",
		"demo",
		0,
		map[string]string{
			"name":   "John",
			"coupon": "RANDOM_CODE",
			"url":    "https://images.google.com",
			"btn1":   "BUTTON1",
			"btn2":   "BUTTON2",
		},
	}

	return r
}

// helper function to print MessageNodes to stdout
func (messageNode *MessageNode) ToString() string {
	var sb strings.Builder

	sb.WriteString(messageNode.Body + "\n")

	for i, text := range messageNode.Button {
		sb.WriteString(fmt.Sprintf("%d %s\n", i+1, text))
	}

	if messageNode.Image != "" {
		sb.WriteString(fmt.Sprintf("img: %s\n", messageNode.Image))
	}

	return sb.String()
}

// helper function to print flows for debugging purposes
func (flow *Flow) ToString() string {
	var sb strings.Builder

	for i := 0; i < len(flow.Nodes); i++ {
		node := flow.Nodes[i]
		sb.WriteString(fmt.Sprintf("node %d\n%s\n", i+1, node.ToString()))
	}

	return sb.String()
}

// example flow
func DemoFlowFactory() *Flow {
	flow := NewFlow("demo")

	welcomeNode := NewMessageNode(
		`Welcome to the demo flow {{.name}}! Are you interested in our coupon promotion?`,
	)

	couponNode := NewMessageNode("Here is our coupon.")

	// maybe use a different type here for target nodes
	goalNode := NewMessageNode(
		"",
	)

	imageNode := NewMessageNode(
		"No worries, have a nice day!",
		WithImage("{{.url}}"),
	)

	welcomeId := AddNode(flow, welcomeNode)
	imageId := AddNode(flow, imageNode)
	couponId := AddNode(flow, couponNode)

	goalId := AddNode(flow, goalNode)

	AddTransition(flow, welcomeId, couponId, "Yes, show me the coupon!")
	AddTransition(flow, welcomeId, imageId, "No, thanks")
	AddTransition(flow, couponId, goalId, "Reveal Coupon")

	return flow
}

// perform substitutions on a template string using values from a map. Map keys
// are present on the template string and are substituted with map values
func PopulateTextFromMap(text string, mapValues map[string]string) (string, error) {
	templ, err := template.New("").Parse(text)

	if err != nil {
		return "", err
	}

	var sb strings.Builder
	err = templ.Execute(&sb, mapValues)

	if err != nil {
		return "", err
	}

	return sb.String(), nil
}

// perform substitutions on a MessageNode using the supplied params
func PopulateNodeWithParams(node *MessageNode, params RequestMessageNodeParams) (*MessageNode, error) {
	body, err := PopulateTextFromMap(node.Body, params)

	if err != nil {
		return nil, err
	}

	buttons := make([]string, 0)
	for _, buttonTemplText := range node.Button {
		buttonText, err := PopulateTextFromMap(buttonTemplText, params)
		buttons = append(buttons, buttonText)

		if err != nil {
			return nil, err
		}
	}

	img, err := PopulateTextFromMap(node.Image, params)

	if err != nil {
		return nil, err
	}

	transitionMap := map[string]int{}

	for msg, nodeId := range node.NextNodes {
		updatedMsg, err := PopulateTextFromMap(msg, params)

		if err != nil {
			return nil, err
		}

		transitionMap[updatedMsg] = nodeId
	}

	return &MessageNode{
		node.NodeId,
		body,
		buttons,
		img,
		transitionMap,
	}, nil
}

// utility function that performs substitutions for a node within a flow and returns it
func PopulateCurrentNode(flow *Flow, nodeId int, params RequestMessageNodeParams) (*MessageNode, error) {
	if flow == nil {
		return nil, errors.New("flow should be non-nil")
	}

	if node, ok := flow.Nodes[nodeId]; ok {
		populatedNode, err := PopulateNodeWithParams(node, params)

		if err != nil {
			return nil, err
		}

		return populatedNode, nil
	}

	return nil, errors.New("invalid node id")

}

// get the next node if there is a valid transition from currentNode to another
// based on message msg
func PopulateNextNode(flow *Flow, currentNode *MessageNode, msg string, params RequestMessageNodeParams) (*MessageNode, error) {
	if nextNodeId, ok := currentNode.NextNodes[msg]; ok {
		return PopulateCurrentNode(flow, nextNodeId, params)
	}

	return nil, errors.New("transition not found")
}

package main

import (
	"chatbot/controllers"
	"chatbot/db"
	"chatbot/stories"
	"database/sql"
	"reflect"
	"testing"
)

// for testing purposes
func DemoRequestStoryData() stories.RequestSendData {
	r := stories.RequestSendData{
		UserId:  "123456",
		Story:   "demo",
		StateId: 0,
		Params: map[string]string{
			"name":   "John",
			"coupon": "RANDOM_CODE",
			"img":    "test.png",
			"btn1":   "BUTTON1",
			"btn2":   "BUTTON2",
		},
	}

	return r
}

func SampleNodeFactory() *stories.MessageNode {
	node := stories.NewMessageNode(
		`Welcome to the demo story {{.name}}! Are you interested in our coupon promotion?`,
		stories.WithButtons("Yes {{.btn1}}", "No {{.btn2}}"),
		stories.WithImage("http://images.test.com/{{.img}}"),
	)

	return node
}

// TODO include more tests for buttons, images etc.
func TestNodeSubstitution(t *testing.T) {
	node := SampleNodeFactory()
	requestStoryData := DemoRequestStoryData()

	type Result struct {
		expected, actual string
	}

	results := []Result{}

	populated, _ := stories.PopulateNodeWithParams(node, requestStoryData.Params)

	results = append(results,
		Result{
			expected: "Welcome to the demo story John! Are you interested in our coupon promotion?",
			actual:   populated.Body,
		},
		Result{
			expected: "Yes BUTTON1",
			actual:   populated.Button[0],
		},
		Result{
			expected: "No BUTTON2",
			actual:   populated.Button[1],
		},
		Result{
			expected: "http://images.test.com/test.png",
			actual:   populated.Image,
		})

	for _, r := range results {
		if r.expected != r.actual {
			t.Errorf("Expected %q but got %q", r.expected, r.actual)
		}
	}
}

func NewMockDatabase() (*sql.DB, error) {
	_db, err := sql.Open("sqlite3", ":memory:")

	if err != nil {
		return nil, err
	}

	// create tables and populate with initial data maybe
	err = db.InitTables(_db)

	if err != nil {
		return nil, err
	}

	err = db.PopulateDB(_db)

	if err != nil {
		return nil, err
	}

	return _db, nil
}

// TODO add tests for story state transitions
func TestStoryTransitions(t *testing.T) {
	_db, err := NewMockDatabase()
	story := stories.DemoStoryFactory()

	if err != nil {
		t.Error(err.Error())
	}

	const username string = "test"
	initStoriestate := &stories.Storiestate{
		Story:  "demo",
		State:  stories.NodeId(0),
		Params: DemoRequestStoryData().Params,
	}

	err = db.InsertStoriestate(_db, username, initStoriestate.Story, initStoriestate.State, initStoriestate.Params)

	if err != nil {
		t.Error(err.Error())
	}

	firstNode, err := controllers.HandleMessageActiveStory(_db, "test", "", true)

	if err != nil {
		t.Error(err.Error())
	}

	populatedNode, err := stories.PopulateCurrentNode(story, stories.NodeId(0), DemoRequestStoryData().Params)

	if err != nil {
		t.Error(err.Error())
	}

	if !reflect.DeepEqual(firstNode, populatedNode) {
		t.Errorf("Expected %q but got %q\n", populatedNode, firstNode)
	}
}

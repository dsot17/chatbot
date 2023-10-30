package controllers

import (
	"chatbot/db"
	"chatbot/stories"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

// return the next MessageNode of the active story for the specified user, if possible,
// according to messageText. When initialMsg is true, the conversation is initiated so
// we just return the current node and ignore the messageText
func HandleMessageActiveStory(_db *sql.DB, username, messageText string, initialMsg bool) (*stories.MessageNode, error) {

	// Get story for user
	currentStoriestate, err := db.GetStoriestateByUserId(_db, username)

	if err != nil {
		log.Println("can't get story state")
		return nil, err
	}

	currentStoryData, err := db.GetStoryDataByName(_db, currentStoriestate.Story)

	if err != nil {
		log.Println("can't get story data")
		return nil, err
	}

	story := &stories.Story{}
	err = json.Unmarshal([]byte(currentStoryData), story)

	if err != nil {
		// try to get a default story
		return nil, err
	}

	currentNode, err := stories.PopulateCurrentNode(story, currentStoriestate.State, currentStoriestate.Params)

	if err != nil {
		return nil, err
	}

	outputNode := currentNode

	if !initialMsg {
		nextNode, err := stories.PopulateNextNode(story, currentNode, messageText, currentStoriestate.Params)

		if err != nil {
			return nil, err

		}
		outputNode = nextNode

		err = db.InsertStoriestate(_db, username, currentStoriestate.Story, outputNode.NodeId, currentStoriestate.Params)

		if err != nil {
			return nil, err
		}
	}

	err = db.InsertStoriestats(_db, username, story.Name, outputNode.NodeId)
	if err != nil {
		log.Println("Failed to log stats to db", err.Error())
	}

	if len(outputNode.NextNodes) == 0 {
		// TODO Reset story to some default value
		log.Printf("Story %s has ended\n", story.Name)
	}

	return outputNode, nil
}

type HttpHandler func(w http.ResponseWriter, r *http.Request)

func HandlerWithDB(db *sql.DB, original func(db *sql.DB, w http.ResponseWriter, r *http.Request)) HttpHandler {
	result := func(w http.ResponseWriter, r *http.Request) {
		original(db, w, r)
	}
	return result
}

// handler that sets active story for a user and initiates the conversation
func MessageHandler(_db *sql.DB, w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request", http.StatusMethodNotAllowed)
		return
	}

	var messageData stories.RequestSendData
	err := json.NewDecoder(r.Body).Decode(&messageData)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Println(messageData)

	user := messageData.UserId
	err = db.InsertStoriestate(_db, user, messageData.Story, messageData.StateId, messageData.Params)
	if err != nil {
		log.Println("Failed to set story state", err.Error())
	}

	msgNode, err := HandleMessageActiveStory(_db, user, "", true)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO use Telegram
	stories.SendToStdout(msgNode)
}

// return json with the number of visits of all states for the
// specified story
func StatsHandler(_db *sql.DB, w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request", http.StatusMethodNotAllowed)
	}

	var stats stories.RequestStats
	err := json.NewDecoder(r.Body).Decode(&stats)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	storiestats, err := db.GetStoriestats(_db, stats.Story)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(storiestats)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

package controllers

import (
	"encoding/json"
	"log"
	"message-bot-demo/db"
	"message-bot-demo/flows"
	"net/http"
)

// return the next MessageNode of the active flow for the specified user, if possible,
// according to messageText. When initialMsg is true, the conversation is initiated so
// we just return the current node and ignore the messageText
func HandleMessageActiveFlow(username, messageText string, initialMsg bool) (*flows.MessageNode, error) {

	// Get flow for user
	currentFlowState, err := db.GetFlowStateByUserId(username)

	if err != nil {
		log.Println("can't get flow state")
		return nil, err
	}

	currentFlowData, err := db.GetFlowDataByName(currentFlowState.Flow)

	if err != nil {
		log.Println("can't get flow data")
		return nil, err
	}

	flow := &flows.Flow{}
	err = json.Unmarshal([]byte(currentFlowData), flow)

	if err != nil {
		// try to get a default flow
		return nil, err
	}

	currentNode, err := flows.PopulateCurrentNode(flow, currentFlowState.State, currentFlowState.Params)

	if err != nil {
		return nil, err
	}

	outputNode := currentNode

	if !initialMsg {
		nextNode, err := flows.PopulateNextNode(flow, currentNode, messageText, currentFlowState.Params)

		if err != nil {
			return nil, err

		}
		outputNode = nextNode

		err = db.InsertFlowState(username, currentFlowState.Flow, outputNode.NodeId, currentFlowState.Params)

		if err != nil {
			return nil, err
		}
	}

	err = db.InsertFlowStats(username, flow.Name, outputNode.NodeId)
	if err != nil {
		log.Println("Failed to log stats to db", err.Error())
	}

	if len(outputNode.NextNodes) == 0 {
		// TODO Reset flow to some default value
		log.Printf("Flow %s has ended\n", flow.Name)
	}

	return outputNode, nil
}

// handler that sets active flow for a user and initiates the conversation
func MessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request", http.StatusMethodNotAllowed)
		return
	}

	var messageData flows.RequestSendData
	err := json.NewDecoder(r.Body).Decode(&messageData)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Println(messageData)

	user := messageData.UserId
	err = db.InsertFlowState(user, messageData.Flow, messageData.StateId, messageData.Params)
	if err != nil {
		log.Println("Failed to set flow state", err.Error())
	}

	msgNode, err := HandleMessageActiveFlow(user, "", true)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO use Telegram
	flows.SendToStdout(msgNode)
}

// return json with the number of visits of all states for the
// specified flow
func StatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request", http.StatusMethodNotAllowed)
	}

	var stats flows.RequestStats
	err := json.NewDecoder(r.Body).Decode(&stats)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	flowStats, err := db.GetFlowStats(stats.Flow)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(flowStats)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

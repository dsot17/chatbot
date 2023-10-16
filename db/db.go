package db

import (
	"database/sql"
	"encoding/json"
	_ "github.com/mattn/go-sqlite3"
	"message-bot-demo/flows"
)

var _db *sql.DB

func Open(path string) error {
	db, err := sql.Open("sqlite3", path)

	if err != nil {
		return err
	}

	_db = db
	return nil
}

func Close() error {
	return _db.Close()
}

func InitTables(path string) error {
	db, err := sql.Open("sqlite3", path)

	if err != nil {
		return err
	}

	err = createFlowsTable(db)
	if err != nil {
		return err
	}

	err = createUsersTable(db)
	if err != nil {
		return err
	}

	err = createStatsTable(db)
	if err != nil {
		return err
	}

	return nil
}

func flowToJSON(flow *flows.Flow) (string, error) {
	jsonText, err := json.Marshal(flow)

	if err != nil {
		return "", err
	}

	return string(jsonText), nil
}

// insert a json representation of the demo flow to the db
func PopulateDB() error {
	demoFlow := flows.DemoFlowFactory()

	jsonFlow, err := flowToJSON(demoFlow)
	if err != nil {
		return err
	}

	err = InsertFlowData("demo", jsonFlow)
	if err != nil {
		return err
	}

	return nil
}

func createFlowsTable(db *sql.DB) error {
	statement := `
CREATE TABLE IF	NOT	EXISTS flows (
	flow TEXT PRIMARY KEY,
	data TEXT
);`
	_, err := db.Exec(statement)
	return err
}

func createUsersTable(db *sql.DB) error {
	statement := `
CREATE TABLE IF	NOT	EXISTS users (
	userId TEXT	PRIMARY	KEY,
	currentFlow	TEXT,
	stateId	INTEGER,
	params TEXT
);`
	_, err := db.Exec(statement)
	return err
}

func createStatsTable(db *sql.DB) error {
	statement := `
CREATE TABLE stats (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	userId TEXT,
	flow TEXT,
	visitedNode	INTEGER,
	FOREIGN	KEY	(userId) REFERENCES	users(userId),
	FOREIGN	KEY	(flow) REFERENCES flows(flow)
);`
	_, err := db.Exec(statement)
	return err
}

// get json data for specified flow
func GetFlowDataByName(flowName string) (string, error) {
	var data string
	statement := `SELECT data FROM flows WHERE flow	= ?`
	err := _db.QueryRow(statement, flowName).Scan(&data)

	if err != nil {
		return "", err
	}

	return data, nil
}

// add a flow to the db
func InsertFlowData(flowName string, data string) error {
	statement := `INSERT INTO flows	(flow, data) VALUES	(?,	?);`
	_, err := _db.Exec(statement, flowName, data)
	return err
}

// get statistics for each state for the given flow
func GetFlowStats(flowName string) (*flows.FlowStats, error) {
	flowStats := &flows.FlowStats{}
	statement := `SELECT visitedNode, COUNT(*) FROM	stats WHERE	flow = ? GROUP BY visitedNode`

	rows, err := _db.Query(statement, flowName)
	defer rows.Close()

	if err != nil {
		return nil, err
	}

	flowStats.Stats = map[int]int{}

	for rows.Next() {
		var visitedNode, count int

		if err := rows.Scan(&visitedNode, &count); err != nil {
			return nil, err
		}

		flowStats.Stats[visitedNode] = count
	}

	return flowStats, nil
}

// mark nodeId as visited for given flow and user
func InsertFlowStats(username string, flowName string, nodeId int) error {
	statement := `INSERT INTO stats	(userId, flow, visitedNode)	VALUES (?, ?, ?);`

	_, err := _db.Exec(statement, username, flowName, nodeId)
	return err
}

// get the active flow and state of the specified user
func GetFlowStateByUserId(userId string) (*flows.FlowState, error) {
	flowState := &flows.FlowState{}
	var params []byte
	statement := `SELECT currentFlow, stateId, params FROM users WHERE userId =	?`

	err := _db.QueryRow(statement, userId).Scan(&flowState.Flow, &flowState.State, &params)

	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(params, &flowState.Params); err != nil {
		return nil, err
	}
	return flowState, nil
}

// modify active flow and state for user
func InsertFlowState(userId string, flowName string, stateId int, params flows.RequestMessageNodeParams) error {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return err
	}

	statement := `INSERT OR	REPLACE	INTO users (userId,	currentFlow, stateId, params) VALUES (?, ?,	?, ?)`
	_, err = _db.Exec(statement, userId, flowName, stateId, string(paramsJSON))
	if err != nil {
		return err
	}

	return nil
}

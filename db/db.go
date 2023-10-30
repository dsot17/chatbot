package db

import (
	"chatbot/stories"
	"database/sql"
	"encoding/json"

	_ "github.com/mattn/go-sqlite3"
)

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)

	if err != nil {
		return nil, err
	}

	return db, nil
}

func InitTables(db *sql.DB) error {
	err := createStoriesTable(db)
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

func storyToJSON(story *stories.Story) (string, error) {
	jsonText, err := json.Marshal(story)

	if err != nil {
		return "", err
	}

	return string(jsonText), nil
}

// insert a json representation of the demo story to the db
func PopulateDB(_db *sql.DB) error {
	demoStory := stories.DemoStoryFactory()

	jsonStory, err := storyToJSON(demoStory)
	if err != nil {
		return err
	}

	err = InsertStoryData(_db, "demo", jsonStory)
	if err != nil {
		return err
	}

	return nil
}

func createStoriesTable(_db *sql.DB) error {
	statement := `
CREATE TABLE IF	NOT	EXISTS stories (
	story TEXT PRIMARY KEY,
	data TEXT
);`
	_, err := _db.Exec(statement)
	return err
}

func createUsersTable(_db *sql.DB) error {
	statement := `
CREATE TABLE IF	NOT	EXISTS users (
	userId TEXT	PRIMARY	KEY,
	currentStory	TEXT,
	stateId	INTEGER,
	params TEXT
);`
	_, err := _db.Exec(statement)
	return err
}

func createStatsTable(_db *sql.DB) error {
	statement := `
CREATE TABLE stats (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	userId TEXT,
	story TEXT,
	visitedNode	INTEGER,
	FOREIGN	KEY	(userId) REFERENCES	users(userId),
	FOREIGN	KEY	(story) REFERENCES stories(story)
);`
	_, err := _db.Exec(statement)
	return err
}

// get json data for specified story
func GetStoryDataByName(_db *sql.DB, storyName string) (string, error) {
	var data string
	statement := `SELECT data FROM stories WHERE story	= ?`
	err := _db.QueryRow(statement, storyName).Scan(&data)

	if err != nil {
		return "", err
	}

	return data, nil
}

// add a story to the db
func InsertStoryData(_db *sql.DB, storyName string, data string) error {
	statement := `INSERT INTO stories	(story, data) VALUES	(?,	?);`
	_, err := _db.Exec(statement, storyName, data)
	return err
}

// get statistics for each state for the given story
func GetStoriestats(_db *sql.DB, storyName string) (*stories.Storiestats, error) {
	storiestats := &stories.Storiestats{}
	statement := `SELECT visitedNode, COUNT(*) FROM	stats WHERE	story = ? GROUP BY visitedNode`

	rows, err := _db.Query(statement, storyName)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	if err != nil {
		return nil, err
	}

	storiestats.Stats = map[int]int{}

	for rows.Next() {
		var visitedNode, count int

		if err := rows.Scan(&visitedNode, &count); err != nil {
			return nil, err
		}

		storiestats.Stats[visitedNode] = count
	}

	return storiestats, nil
}

// mark nodeId as visited for given story and user
func InsertStoriestats(_db *sql.DB, username string, storyName string, nodeId stories.NodeId) error {
	statement := `INSERT INTO stats	(userId, story, visitedNode)	VALUES (?, ?, ?);`

	_, err := _db.Exec(statement, username, storyName, nodeId)
	return err
}

// get the active story and state of the specified user
func GetStoriestateByUserId(_db *sql.DB, userId string) (*stories.Storiestate, error) {
	storiestate := &stories.Storiestate{}
	var params []byte
	statement := `SELECT currentStory, stateId, params FROM users WHERE userId =	?`

	err := _db.QueryRow(statement, userId).Scan(&storiestate.Story, &storiestate.State, &params)

	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(params, &storiestate.Params); err != nil {
		return nil, err
	}
	return storiestate, nil
}

// modify active story and state for user
func InsertStoriestate(_db *sql.DB, userId string, storyName string, stateId stories.NodeId, params stories.RequestMessageNodeParams) error {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return err
	}

	statement := `INSERT OR	REPLACE	INTO users (userId,	currentStory, stateId, params) VALUES (?, ?,	?, ?)`
	_, err = _db.Exec(statement, userId, storyName, stateId, string(paramsJSON))
	if err != nil {
		return err
	}

	return nil
}

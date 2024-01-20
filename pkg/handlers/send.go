package handlers

import (
	"ap-server/pkg/app"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

func SendHandler(w http.ResponseWriter, r *http.Request) {
	// parse request and verify API key for account
	if err := r.ParseForm(); err != nil {
        http.Error(w, "Error parsing the form for send", http.StatusBadRequest)
        return
    }
    key := r.FormValue("apikey")
	name := r.FormValue("acct")
	msg := r.FormValue("message")
	
	matched, err := checkAPIKey(key, name)
	if err != nil || !matched {
        http.Error(w, "API key error", http.StatusBadRequest)
        return
    }

	// send message to all followers and add to messages database
	sendMessageToFollowers(msg, name, w)

	// respond
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"msg": "ok"})
}


func checkAPIKey(key string, name string) (bool, error) {
	db := app.App.DB
	domain := app.App.Domain
	dbName := fmt.Sprintf("%s@%s", name, domain)
	row := db.QueryRow("SELECT apikey FROM accounts WHERE name = ?", dbName)
	
	var dbKey string
	err := row.Scan(&dbKey)
	if err != nil {
		return false, err
	}
	return dbKey == key, nil
}


func sendMessageToFollowers(msg string, name string, w http.ResponseWriter) {
	followers := getFollowers(w, name)
	if len(followers) == 0 {
		http.Error(w, "No followers found", http.StatusBadRequest)
		return
	}
	for _, follower := range followers {
		// get the note object
		guidNote := createGuid()
		noteObj := getNoteObj(guidNote, msg, name)
		// get the create object for the note's create activity
		guidCreate := createGuid()
		createObj := getCreateObj(guidCreate, name, follower, noteObj)

		// add both objects' json str into messages database
		noteJSONStr, _ := json.Marshal(noteObj)
		createJSONStr, _ := json.Marshal(createObj)
		db := app.App.DB
		stmt, _ := db.Prepare("INSERT OR REPLACE INTO messages(guid, message) VALUES(?, ?)")
		_, err := stmt.Exec(guidNote, noteJSONStr)
		if err != nil {
			http.Error(w, "Error adding message", http.StatusInternalServerError)
		}
		_, err = stmt.Exec(guidCreate, createJSONStr)
		if err != nil {
			http.Error(w, "Error adding message", http.StatusInternalServerError)
		}

		// sign and send the create activity
		oppInbox := follower + "/inbox"
		actorUrl, _ := url.Parse(follower)
		oppDomain := actorUrl.Hostname()
		signAndSendMsg(w, oppInbox, oppDomain, createJSONStr, name, app.App.Domain)
	}
}


func getNoteObj(guid string, msg string, name string) Note {
	return Note{
		ID:           fmt.Sprintf("https://%s/m/%s", app.App.Domain, guid),
		Type:         "Note",
		Published:    time.Now().UTC().Format(http.TimeFormat),
		AttributedTo: fmt.Sprintf("https://%s/u/%s", app.App.Domain, name),
		Content:      msg,
		To:           []string{"https://www.w3.org/ns/activitystreams#Public"},
	}
}


func getCreateObj(guid string, name string, follower string, noteObj Note) CreateActivity {
	return CreateActivity{
		Context:      "https://www.w3.org/ns/activitystreams",
		ID:           fmt.Sprintf("https://%s/m/%s", app.App.Domain, guid),
		Type:         "Create",
		Actor:        fmt.Sprintf("https://%s/u/%s", app.App.Domain, name),
		To:           []string{"https://www.w3.org/ns/activitystreams#Public"},
		CC:           []string{follower},
		Object:       noteObj,
	}
}


type Note struct {
    ID            string   `json:"id"`
    Type          string   `json:"type"`
    Published     string   `json:"published"`
    AttributedTo  string   `json:"attributedTo"`
    Content       string   `json:"content"`
    To            []string `json:"to"`
}


type CreateActivity struct {
    Context       string      `json:"@context"`
    ID            string      `json:"id"`
    Type          string      `json:"type"`
    Actor         string      `json:"actor"`
    To            []string    `json:"to"`
    CC            []string    `json:"cc"`
    Object        Note `json:"object"`
}
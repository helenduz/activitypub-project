package handlers

import (
	"ap-server/pkg/app"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func UserNameHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
    name := vars["name"]
	if name == "" {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	db := app.App.DB
	domain := app.App.Domain
	// get the user's actor record from db
	dbName := fmt.Sprintf("%s@%s", name, domain)
	row := db.QueryRow("SELECT actor FROM accounts WHERE name = ?", dbName)
	
	var actorJSONStr []byte
	err := row.Scan(&actorJSONStr)
	if err != nil { // handles no record found as well
		handleErr(err, w, name) // defined in webfinger.go
		return
	}
	// send result
	w.Header().Set("Content-Type", "application/json")
	w.Write(actorJSONStr)
}


func UserFollowersHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
    name := vars["name"]
	if name == "" {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// get the followers from db
	// in db, followers is stored as a JSON string of the list of each follower's username, need to convert it to a followersCollection for response
	followers := getFollowers(w, name)
	domain := app.App.Domain
	followersCollectionObj := getFollowersCollectionObj(name, domain, followers)

	// send result
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(followersCollectionObj)

}


func getFollowers(w http.ResponseWriter, name string) []string {
	db := app.App.DB
	domain := app.App.Domain
	dbName := fmt.Sprintf("%s@%s", name, domain)
	row := db.QueryRow("SELECT followers FROM accounts WHERE name = ?", dbName)
	
	var followersJSONStr []byte
	err := row.Scan(&followersJSONStr)
	if err != nil { // handles no record found as well
		handleErr(err, w, name) // defined in webfinger.go
		return nil
	}
	var followers []string
    json.Unmarshal(followersJSONStr, &followers)
	if len(string(followersJSONStr)) == 0 {
		followers = make([]string, 0)
	} // deal with case where followers row is NULL/not initialized (new account)
	return followers
}


func getFollowersCollectionObj(name string, domain string, followers []string) FollowersCollection {
	return FollowersCollection{
		Type: "OrderedCollection",
		TotalItems: len(followers),
		ID: fmt.Sprintf("https://%s/u/%s/followers", domain, name),
		First: First{
			Type: "OrderedCollectionPage",
			TotalItems: len(followers),
			PartOf: fmt.Sprintf("https://%s/u/%s/followers", domain, name),
			OrderedItems: followers,
			ID: fmt.Sprintf("https://%s/u/%s/followers?page=1", domain, name),
		},
		Context: []string{
            "https://www.w3.org/ns/activitystreams",
        },
	}
}


type FollowersCollection struct {
    Type       string `json:"type"`
    TotalItems int    `json:"totalItems"`
    ID         string `json:"id"`
    First      First `json:"first"`
    Context []string `json:"@context"`
}


type First struct {
	Type        string   `json:"type"`
	TotalItems  int      `json:"totalItems"`
	PartOf      string   `json:"partOf"`
	OrderedItems []string `json:"orderedItems"`
	ID          string   `json:"id"`
}
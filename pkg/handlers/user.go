package handlers

import (
	"ap-server/pkg/app"
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
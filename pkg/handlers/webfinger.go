package handlers

import (
	"ap-server/pkg/app"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
)

func WebfingerHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
    resource := queryParams.Get("resource")
	if len(resource) == 0 || !strings.HasPrefix(resource, "acct:") {
		http.Error(w, "Bad request. Please make sure acct:USER@DOMAIN is what you are sending as the 'resource' query parameter.", http.StatusBadRequest)
		return
	}
	// find webfinger record for account in db
	name := strings.Replace(resource, "acct:", "", 1)
	db := app.App.DB
	row := db.QueryRow("SELECT webfinger FROM accounts WHERE name = ?", name)
	
	var webfingerJSONStr []byte
	err := row.Scan(&webfingerJSONStr)
	if err != nil { // handles no record found as well
		handleErr(err, w, name)
		return
	}
	// send result
	w.Header().Set("Content-Type", "application/json")
	w.Write(webfingerJSONStr)
}

func handleErr(err error, w http.ResponseWriter, name string) {
	if err == sql.ErrNoRows {
		http.Error(w, fmt.Sprintf("No record found for %s", name), http.StatusNotFound)
	} else {
		log.Printf("Error in getting webfinger record for account %s", name)
		log.Println(err)
		http.Error(w, "Error in getting webfinger record", http.StatusInternalServerError)
	}
}
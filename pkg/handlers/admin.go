package handlers

import (
	"ap-server/pkg/utils"
	"fmt"
	"net/http"
)

func CreateHandler(w http.ResponseWriter, r *http.Request) {
	// parse body
	if err := r.ParseForm(); err != nil {
        http.Error(w, "Error parsing the form", http.StatusInternalServerError)
        return
    }
    name := r.FormValue("account")
	fmt.Println("hello ", name)

	// create keypair
	privKey, pubKey := utils.GetEncodedKeys()

    fmt.Println("Private Key:", privKey)
    fmt.Println("Public Key:", pubKey)

	// create actor, webfinger, and api key fields for account
	
	// insert to db
}
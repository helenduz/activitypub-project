package handlers

import (
	"ap-server/pkg/app"
	"ap-server/pkg/utils"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func CreateHandler(w http.ResponseWriter, r *http.Request) {
	// parse body
	if err := r.ParseForm(); err != nil {
        http.Error(w, "Error parsing the form", http.StatusInternalServerError)
        return
    }
    name := r.FormValue("account")

	// create keypair
	privKey, pubKey := utils.GetEncodedKeys()
	
	domain := app.App.Domain
	db := app.App.DB
	// create actor, webfinger, and api key fields for account
	actorJSONStr, _ := json.Marshal(getActorObj(name, domain, pubKey))
	webfingerJSONStr, _ := json.Marshal(getWebFingerObj(name, domain))
	apiKey := createAPIKey()

	// insert to db
	stmt, _ := db.Prepare("INSERT or REPLACE into accounts(name, actor, apikey, pubkey, privkey, webfinger) values(?, ?, ?, ?, ?, ?)")
	dbName := fmt.Sprintf("%s@%s", name, domain)
	_, err := stmt.Exec(dbName, actorJSONStr, apiKey, pubKey, privKey, webfingerJSONStr)

	// response
	if err != nil {
		http.Error(w, "Error creating account", http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"apikey": apiKey, "msg": "ok"}) // send API key
	}
}

// creates a random 32 character HEX string
func createAPIKey() string {
	b := make([]byte, 16) // 16 bytes, 32 HEX chars
	_, err := rand.Read(b)
	if err != nil {
		log.Fatalln("Error creating API key", err)
	}
	hexString := hex.EncodeToString(b)
	return hexString
}

func getActorObj(name string, domain string, pubKey string) Actor {
    idURI := fmt.Sprintf("https://%s/u/%s", domain, name)
    return Actor{
        Context: []string{
            "https://www.w3.org/ns/activitystreams",
            "https://w3id.org/security/v1",
        },
        ID:                idURI,
        Type:              "Person",
        PreferredUsername: name,
        Inbox:             fmt.Sprintf("https://%s/api/inbox", domain),
        Outbox:            idURI + "/outbox",
        Followers:         idURI + "/followers",
        PublicKey: PublicKey{
            ID:           idURI + "#main-key",
            Owner:        idURI,
            PublicKeyPem: pubKey,
        },
    }
}

func getWebFingerObj(name string, domain string) Webfinger {
    return Webfinger{
        Subject: fmt.Sprintf("acct:%s@%s", name, domain),
        Links: []Link{
            {
                Rel:  "self",
                Type: "application/activity+json",
                Href: fmt.Sprintf("https://%s/u/%s", domain, name),
            },
        },
    }
}

type PublicKey struct {
    ID           string `json:"id"`
    Owner        string `json:"owner"`
    PublicKeyPem string `json:"publicKeyPem"`
}

type Actor struct {
    Context           []string `json:"@context"`
    ID                string   `json:"id"`
    Type              string   `json:"type"`
    PreferredUsername string   `json:"preferredUsername"`
    Inbox             string   `json:"inbox"`
    Outbox            string   `json:"outbox"`
    Followers         string   `json:"followers"`
    PublicKey         PublicKey `json:"publicKey"`
}

type Link struct {
    Rel  string `json:"rel"`
    Type string `json:"type"`
    Href string `json:"href"`
}

type Webfinger struct {
    Subject string `json:"subject"`
    Links   []Link `json:"links"`
}
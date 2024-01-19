package handlers

import (
	"ap-server/pkg/app"
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/exp/slices"
)

// note: currently only handles Follow acvitivies!
func InboxHandler(w http.ResponseWriter, r *http.Request) {
	// parse the follow activity object in request
	var followObj FollowActivity
    err := json.NewDecoder(r.Body).Decode(&followObj)
    if err != nil {
        http.Error(w, "Error parsing body", http.StatusBadRequest)
        return
    }

	// ignore non-follow requests
	if (followObj.Type != "Follow") {
		return // returns 200 OK
	}
	
	// check if user exists
	myDomain := app.App.Domain
	myName := strings.Replace(followObj.Object, fmt.Sprintf("https://%s/u/", myDomain), "", 1)
	err = checkUserExists(myName)
	if err != nil { // handles no record found as well
		handleErr(err, w, myName) // defined in webfinger.go
		return
	}
	
	// send accept message
	oppInbox := followObj.Actor + "/inbox"
	actorUrl, _ := url.Parse(followObj.Actor)
	oppDomain := actorUrl.Hostname()
	acceptObj := getAcceptObj(myName, myDomain, followObj)
	acceptJSONStr, err := json.Marshal(acceptObj)
	if err != nil {
		log.Fatalln(err)
	}
	signAndSendMsg(w, oppInbox, oppDomain, acceptJSONStr, myName, myDomain)

	// update followers in db
	followers := getFollowers(w, myName)
	if !slices.Contains(followers, followObj.Actor) {
		followers = append(followers, followObj.Actor)
	}
	followersJSONStr, _ := json.Marshal(followers)
	db := app.App.DB
	dbName := fmt.Sprintf("%s@%s", myName, myDomain)
	stmt, _ := db.Prepare("UPDATE accounts SET followers=? WHERE name=?")
	_, err = stmt.Exec(followersJSONStr, dbName)
	if err != nil {
		log.Println("Updating followers in db: ", err)
		http.Error(w, "Error when handling request", http.StatusBadRequest)
		return
	}
	fmt.Println("Updated followers to: ", followers)
}


func checkUserExists(name string) error {
	db := app.App.DB
	dbName := fmt.Sprintf("%s@%s", name, app.App.Domain)
	row := db.QueryRow("SELECT actor FROM accounts WHERE name = ?", dbName)
	
	var actorJSONStr []byte
	err := row.Scan(&actorJSONStr)
	if err != nil {
		return err
	}
	return nil
}


func signAndSendMsg(w http.ResponseWriter, oppInbox string, oppDomain string, msgJSONStr []byte, myName string, myDomain string) {
	// getting info we need
	oppInboxFragment := strings.Replace(oppInbox, "https://" + oppDomain, "", 1)
	privKey, err := getPrivKey(myName)
	if err != nil {
		handleErr(err, w, myName)
		return
	}
	privKeyRSA := parsePrivKeyPEM([]byte(privKey))

	// create digest hash (hash json of message->encode to base 64)
	hashedMsg := sha256.Sum256(msgJSONStr)
	digestHash := base64.StdEncoding.EncodeToString(hashedMsg[:])

	// prepare the string to sign
	d := time.Now().UTC().Format(http.TimeFormat)
	stringToSign := fmt.Sprintf("(request-target): post %s\nhost: %s\ndate: %s\ndigest: SHA-256=%s", oppInboxFragment, oppDomain, d, digestHash)

	// signing the overall string (hash->sign with privKey->encode to base 64)
	hashedStringToSign := sha256.Sum256([]byte(stringToSign))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privKeyRSA, crypto.SHA256, hashedStringToSign[:])
	if err != nil {
		log.Fatalln(err)
	}
	signatureB64 := base64.StdEncoding.EncodeToString(signature)

	// forming the header signature
	headerSig := fmt.Sprintf("keyId=\"https://%s/u/%s#main-key\",headers=\"(request-target) host date digest\",signature=\"%s\"", myDomain, myName, signatureB64)
	
	// make HTTP request for follower's inbox & log response
	go sendToHTTP(oppInbox, oppDomain, d, digestHash, headerSig, msgJSONStr)
}


func getAcceptObj(myName string, myDomain string, followObj FollowActivity) AcceptActivity {
	guid := createGuid()
	return AcceptActivity{
		Context: "https://www.w3.org/ns/activitystreams",
		Id: fmt.Sprintf("https://%s/%s", myDomain, guid),
		Type: "Accept",
		Actor: fmt.Sprintf("https://%s/u/%s", myDomain, myName),
		Object: followObj,
	}
}


func getPrivKey(name string) (string, error) {
	db := app.App.DB
	dbName := fmt.Sprintf("%s@%s", name, app.App.Domain)
	row := db.QueryRow("SELECT privkey FROM accounts WHERE name = ?", dbName)
	
	var key string
	err := row.Scan(&key)
	if err != nil {
		return "", err
	}
	return key, nil
}


func createGuid() string {
	b := make([]byte, 16) // 16 bytes, 32 HEX chars
	_, err := rand.Read(b)
	if err != nil {
		log.Fatalln("Error creating message guid", err)
	}
	hexString := hex.EncodeToString(b)
	return hexString
}


func parsePrivKeyPEM(privKeyPEM []byte) (*rsa.PrivateKey) {
    block, _ := pem.Decode(privKeyPEM)
    if block == nil {
		log.Fatalln(errors.New("failed to parse PEM block containing the key"))
    }
    privKeyRSA, err := x509.ParsePKCS1PrivateKey(block.Bytes)
    if err != nil {
        log.Fatalln(errors.New("failed to parse PEM block into rsa private key"))
    }

    return privKeyRSA
}


func sendToHTTP(oppInbox string, oppDomain string, d string, digestHash string, headerSig string, msgJSONStr []byte) {
    req, _ := http.NewRequest("POST", oppInbox, bytes.NewBuffer(msgJSONStr))
    req.Header.Set("Host", oppDomain)
    req.Header.Set("Date", d)
    req.Header.Set("Digest", "SHA-256="+digestHash)
    req.Header.Set("Signature", headerSig)
    req.Header.Set("Content-Type", "application/json")
    resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
    defer resp.Body.Close()
    body, _ := io.ReadAll(resp.Body)
    fmt.Printf("Response to sending msg: STATUS %s, BODY %s\n", resp.Status, string(body))
}


type FollowActivity struct {
	Context string `json:"@context"`
	Actor string `json:"actor"`
	Type string `json:"type"`
	Object string `json:"object"`
	Id string `json:"id"`
}

type AcceptActivity struct {
	Context string `json:"@context"`
	Actor string `json:"actor"`
	Type string `json:"type"`
	Object FollowActivity `json:"object"`
	Id string `json:"id"`
}
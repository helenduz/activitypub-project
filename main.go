package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	app "ap-server/pkg/app"
	handlers "ap-server/pkg/handlers"
	middlewares "ap-server/pkg/middlewares"

	_ "github.com/mattn/go-sqlite3" // blank import for db initialization

	"github.com/gorilla/mux"
	"github.com/henvic/httpretty"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func catchAllHandler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("<h1>Testing!</h1>"))
}

func dbSetUp() *sql.DB {
	os.Remove("./ap-server.db")
	db, err := sql.Open("sqlite3", "./ap-server.db")
	if err != nil {
		log.Fatal(err)
	}

	sqlStmt := `CREATE TABLE IF NOT EXISTS accounts (name TEXT PRIMARY KEY, privkey TEXT, pubkey TEXT, webfinger TEXT, actor TEXT, apikey TEXT, followers TEXT, messages TEXT)`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("%q: %s\n", err, sqlStmt)
	}
	sqlStmt = `CREATE TABLE IF NOT EXISTS messages (guid TEXT PRIMARY KEY, message TEXT)`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("%q: %s\n", err, sqlStmt)
	}

	return db
}

func main() {
	// set up port number
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// set up db
	db := dbSetUp()
	defer db.Close()

	// register server resources so packages can access them
	app.InitApp(db, os.Getenv("DOMAIN"), os.Getenv("PORT"))

	// set up routes
	// main router
	r := mux.NewRouter().StrictSlash(true)
	// home page route
	fs := http.FileServer(http.Dir("static"))
	r.Handle("/admin/", http.StripPrefix("/admin/", fs))

	defaultCors := cors.Default().Handler

	// webfinger route
	webfingerSubrouter := r.PathPrefix("/.well-known/webfinger").Subrouter()
	webfingerSubrouter.Use(defaultCors)
	webfingerSubrouter.PathPrefix("").HandlerFunc(handlers.WebfingerHandler).Methods("GET")

	// /u routes
	userSubrouter := r.PathPrefix("/u").Subrouter()
	userSubrouter.Use(defaultCors)
	userSubrouter.HandleFunc("/{name}/followers", handlers.UserFollowersHandler).Methods("GET")
	userSubrouter.HandleFunc("/{name}", handlers.UserNameHandler).Methods("GET")

	// inbox route
	inboxSubrouter := r.PathPrefix("/api/inbox").Subrouter()
	inboxSubrouter.Use(defaultCors)
	inboxSubrouter.PathPrefix("").HandlerFunc(handlers.InboxHandler).Methods("POST")

	// send msg route
	sendSubrouter := r.PathPrefix("/api/send").Subrouter()
	sendSubrouter.Use(defaultCors)
	sendSubrouter.PathPrefix("").HandlerFunc(handlers.SendHandler).Methods("POST")

	// credentials cors + http authorizer subroute (/api/admin)
	// set up http authorizer
	credentialCors := cors.New(cors.Options{
        AllowCredentials: true,
        AllowedOrigins:   []string{"*"},
        AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowedHeaders:   []string{"Authorization", "Content-Type"},
        AllowOriginFunc: func(origin string) bool {
            return true
        },
    })
	adminSubrouter := r.PathPrefix("/api/admin").Subrouter()
	adminSubrouter.Use(credentialCors.Handler)
	adminSubrouter.Use(middlewares.BasicAuthMiddleware)
	adminSubrouter.HandleFunc("/create", handlers.CreateHandler).Methods("POST")

	// catch-all route
	r.PathPrefix("/").HandlerFunc(catchAllHandler)

	// set up server logging
	logger := &httpretty.Logger{
		RequestBody:    true,
		ResponseHeader: true,
		ResponseBody:   true,
		Formatters:     []httpretty.Formatter{&httpretty.JSONFormatter{}},
	}

	// start server
	fmt.Printf("Listening on port %s\n", port)
	http.ListenAndServe(":"+port, logger.Middleware(r))
}
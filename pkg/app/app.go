package app

import "database/sql"

// define a struct and var for packages to access server resources
type Server struct {
    DB *sql.DB
	Domain string
	Port string
}

var App *Server

func InitApp(db *sql.DB, domain string, port string) {
	App = &Server {
		DB: db,
		Domain: domain,
		Port: port,
	}
}
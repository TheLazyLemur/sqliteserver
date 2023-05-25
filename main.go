package main

import (
	"database/sql"
	"flag"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/thelazylemur/sqliteserver/server"
)

func main() {
	port := os.Getenv("PORT")
	if len(port) == 0 || port == " " {
		port = "5050"
	}

	s := &server.Server{
		Port:      ":" + port,
		Followers: make(map[string]struct{}),
	}

	dbAddr := ""
	flag.StringVar(&dbAddr, "dbAddr", "./database.db", "Db addr")
	flag.Parse()

	db, err := sql.Open("sqlite3", dbAddr)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	s.Db = db

	s.Run()
}

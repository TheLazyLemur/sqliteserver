package main

import (
	"database/sql"
	"flag"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/thelazylemur/sqliteserver/server"
)

func initLogTable(logDb *sql.DB) {
	query := `CREATE TABLE IF NOT EXISTS log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		query TEXT
	)`

	_, err := logDb.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "5050"
	}

	logDb, err := sql.Open("sqlite3", "./logs.db")
	if err != nil {
		log.Fatal(err)
	}
	defer logDb.Close()
	initLogTable(logDb)

	s := &server.Server{
		Port:  ":" + port,
		Logs:  []string{},
		LogDb: logDb,
	}

	dbAddr := flag.String("dbAddr", "./database.db", "Database address")
	flag.Parse()

	db, err := sql.Open("sqlite3", *dbAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	s.Db = db

	s.Run()
}

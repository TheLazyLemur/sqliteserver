package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/thelazylemur/sqliteserver/server"
	"github.com/thelazylemur/sqliteserver/types"
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
	flag.BoolVar(&s.IsLeader, "leader", true, "is leader")
	flag.StringVar(&s.LeaderAddrr, "leaderAddr", "", "leader address")
	flag.StringVar(&dbAddr, "dbAddr", "./database.db", "Db addr")
	flag.Parse()

	db, err := sql.Open("sqlite3", dbAddr)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	s.Db = db

	if !s.IsLeader {
		r := new(types.AddFollowerRequest)

		r.Address = "http://localhost"
		r.Port = s.Port

		rJson, err := json.Marshal(r)
		if err != nil {
			log.Fatal(err)
		}

		req, _ := http.NewRequest("POST", s.LeaderAddrr+"/follower", io.NopCloser(bytes.NewBuffer(rJson)))
		req.Header.Add("secret", "secret")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			log.Fatal(resp.StatusCode)
		}
	}

	s.Run()
}

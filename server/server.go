package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/thelazylemur/sqliteserver/server/middleware"
	"github.com/thelazylemur/sqliteserver/types"
)

type Server struct {
	Db          *sql.DB
	Port        string
	IsLeader    bool
	Followers   map[string]struct{}
	LeaderAddrr string
}

func (s *Server) Run() {
	router := chi.NewRouter()
	router.Use(middleware.DbPassword)
	router.Post("/query", s.QueryHandler)
	router.Post("/health", s.HealthHandler)
	router.Post("/follower", s.FollowerHandler)

	log.Printf("Listening on port %s", s.Port)
	log.Fatal(http.ListenAndServe(s.Port, router))
}

func (s *Server) FollowerHandler(w http.ResponseWriter, r *http.Request) {
	addFollowerReq := new(types.AddFollowerRequest)
	err := json.NewDecoder(r.Body).Decode(addFollowerReq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Println(addFollowerReq.Address + addFollowerReq.Port)

	s.Followers[addFollowerReq.Address+addFollowerReq.Port] = struct{}{}
}

func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (s *Server) QueryHandler(w http.ResponseWriter, r *http.Request) {
	queryResult := new(types.QueryResult)

	var query types.Query
	err := json.NewDecoder(r.Body).Decode(&query)
	if err != nil {
		queryResult.Error = err.Error()
		queryResult.Result = nil
		json.NewEncoder(w).Encode(queryResult)
		fmt.Println(err)
		return
	}
	fmt.Println(query.SqlQuery)

	rows, err := s.Db.Query(query.SqlQuery)
	if err != nil {
		queryResult.Error = err.Error()
		queryResult.Result = nil
		json.NewEncoder(w).Encode(queryResult)
		return
	}

	defer rows.Close()

	results := []map[string]interface{}{}

	columns, err := rows.Columns()
	if err != nil {
		queryResult.Error = err.Error()
		queryResult.Result = nil
		json.NewEncoder(w).Encode(queryResult)
		return
	}
	for rows.Next() {
		columnPointers := make([]interface{}, len(columns))
		columnValues := make([]interface{}, len(columns))

		for i := range columnValues {
			columnPointers[i] = &columnValues[i]
		}

		err := rows.Scan(columnPointers...)
		if err != nil {
			queryResult.Error = err.Error()
			queryResult.Result = nil
			json.NewEncoder(w).Encode(queryResult)
			return
		}

		row := make(map[string]interface{})
		for i, colName := range columns {
			val := columnValues[i]
			row[colName] = val
		}

		results = append(results, row)
	}

	if err = rows.Err(); err != nil {
		queryResult.Error = err.Error()
		queryResult.Result = nil
		json.NewEncoder(w).Encode(queryResult)
		return
	}

	if strings.Contains(query.SqlQuery, "INSERT") || strings.Contains(query.SqlQuery, "UPDATE") || strings.Contains(query.SqlQuery, "DELETE") || strings.Contains(query.SqlQuery, "CREATE") {
		fmt.Println("Sending to followers")
		//TODO: handle replication
		s.SendToFollowers(query)
	}

	queryResult.Result = results
	json.NewEncoder(w).Encode(queryResult)

	_ = json.NewEncoder(w).Encode(queryResult)
}

func (s *Server) SendToFollowers(query types.Query) {
	//TODO: handle replication
	jsonToSend, err := json.Marshal(query)
	if err != nil {
		log.Print(err)
	}

	for ip := range s.Followers {
		client := &http.Client{}
		req, err := http.NewRequest("POST", ip+"/query", bytes.NewBuffer(jsonToSend))
		if err != nil {
			log.Fatal(err)
		}
		req.Header.Set("secret", "secret")
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
	}
}

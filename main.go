package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi"
	_ "github.com/mattn/go-sqlite3"
)

type Server struct {
	db        *sql.DB
	port      string
	isLeader  bool
	followers map[string]struct{}
}

type Query struct {
	SqlQuery string `json:"sqlQuery"`
}

type QueryResult struct {
	Result []map[string]interface{} `json:"result"`
	Error  string                   `json:"error"`
}

func DbPassword(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secret := r.Header.Get("secret")
		if secret != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) Run() {
	router := chi.NewRouter()
	router.Use(DbPassword)
	router.Post("/query", s.QueryHandler)
	router.Post("/health", s.HealthHandler)
	log.Fatal(http.ListenAndServe(s.port, router))
}

func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (s *Server) QueryHandler(w http.ResponseWriter, r *http.Request) {
	queryResult := new(QueryResult)

	var query Query
	err := json.NewDecoder(r.Body).Decode(&query)
	if err != nil {
		queryResult.Error = err.Error()
		queryResult.Result = nil
		json.NewEncoder(w).Encode(queryResult)
		return
	}
	fmt.Println(query.SqlQuery)

	rows, err := s.db.Query(query.SqlQuery)
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

	if strings.Contains(query.SqlQuery, "INSERT") {
		//TODO: handle replication
		s.SendToFollowers(query)
	}

	queryResult.Result = results
	json.NewEncoder(w).Encode(queryResult)

	_ = json.NewEncoder(w).Encode(queryResult)
}

func (s *Server) SendToFollowers(query Query) {
	//TODO: handle replication
	jsonToSend, err := json.Marshal(query)
	if err != nil {
		log.Print(err)
	}

	for ip := range s.followers {
		client := &http.Client{}
		req, err := http.NewRequest("POST", ip, bytes.NewBuffer(jsonToSend))
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

func main() {
	db, err := sql.Open("sqlite3", "./database.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	s := &Server{
		port:      ":" + os.Getenv("PORT"),
		db:        db,
		isLeader:  true,
		followers: make(map[string]struct{}),
	}

	s.Run()
}

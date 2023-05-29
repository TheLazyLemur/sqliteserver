package server

import (
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
	Db    *sql.DB
	LogDb *sql.DB
	Port  string
	Logs  []string
}

func (s *Server) Run() {
	router := chi.NewRouter()
	router.Use(middleware.DbPassword)
	router.Post("/query", s.QueryHandler)
	router.Post("/health", s.HealthHandler)

	log.Printf("Listening on port %s\n", s.Port)
	log.Fatal(http.ListenAndServe(s.Port, router))
}

func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func ShouldAddQueryToLogs(query types.Query) bool {
	lowerdQuery := strings.ToLower(query.SqlQuery)
	return strings.Contains(lowerdQuery, "insert") || strings.Contains(lowerdQuery, "update") || strings.Contains(lowerdQuery, "delete")
}

func (s *Server) QueryHandler(w http.ResponseWriter, r *http.Request) {
	queryResult := new(types.QueryResult)

	var query types.Query
	err := json.NewDecoder(r.Body).Decode(&query)
	if err != nil {
		queryResult.Error = err.Error()
		queryResult.Result = nil
		_ = json.NewEncoder(w).Encode(queryResult)
		fmt.Println(err)
		return
	}
	fmt.Println(len(query.Params))

	rows, err := s.Db.Query(query.SqlQuery, query.Params...)
	if err != nil {
		queryResult.Error = err.Error()
		queryResult.Result = nil
		_ = json.NewEncoder(w).Encode(queryResult)
		return
	}

	defer rows.Close()

	results := []map[string]interface{}{}

	columns, err := rows.Columns()
	if err != nil {
		queryResult.Error = err.Error()
		queryResult.Result = nil
		_ = json.NewEncoder(w).Encode(queryResult)
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
			_ = json.NewEncoder(w).Encode(queryResult)
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
		_ = json.NewEncoder(w).Encode(queryResult)
		return
	}

	if ShouldAddQueryToLogs(query) {
		_, err := s.LogDb.Exec("INSERT INTO log (query) VALUES (?)", query.SqlQuery)
		if err != nil {
			queryResult.Error = err.Error()
			queryResult.Result = nil
			_ = json.NewEncoder(w).Encode(queryResult)
			return
		}

		s.Logs = append(s.Logs, query.SqlQuery)
		for _, log := range s.Logs {
			fmt.Println(log)
		}
	}

	queryResult.Result = results
	_ = json.NewEncoder(w).Encode(queryResult)
}

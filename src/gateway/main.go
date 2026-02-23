package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/zukigit/chat/src/gateway/handlers"
	db "github.com/zukigit/chat/src/lib/db"
)

func newRouter(q *db.Queries) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/login", handlers.LoginHandler(q)).Methods(http.MethodPost)
	return r
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://chat:changeme@localhost:5432/chat?sslmode=disable"
	}

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}

	q := db.New(sqlDB)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      newRouter(q),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("Gateway listening on :8080")
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

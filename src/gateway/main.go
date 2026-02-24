package main

import (
	"database/sql"
	"fmt"
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

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	host := getenv("DB_HOST", "localhost")
	port := getenv("DB_PORT", "5432")
	user := getenv("DB_USER", "chat")
	password := os.Getenv("DB_PASSWORD")
	dbname := getenv("DB_NAME", "chat")
	sslmode := getenv("DB_SSLMODE", "disable")

	if password == "" {
		log.Fatal("DB_PASSWORD env variable is required but not set")
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, dbname, sslmode)

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

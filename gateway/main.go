package main

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/zukigit/chat/gateway/internal/handlers"
	"github.com/zukigit/chat/gateway/internal/lib"
)

func newRouter() *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/login", handlers.LoginHandler).Methods("POST")
	router.HandleFunc("/signup", handlers.SignupHandler).Methods("POST")

	return router
}

func main() {
	server := &http.Server{
		Addr:         lib.Getenv("CHAT_GATEWAY_LISTEN_PORT", ":1122"),
		Handler:      newRouter(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	publicIP := lib.GetPublicIP()
	lib.InfoLog.Println("started listening..")
	lib.InfoLog.Printf("public IP: %s:%s\n", publicIP, "1122")

	if err := server.ListenAndServe(); err != nil {
		lib.ErrorLog.Fatalf("could not listen, err: %v", err)
	}
}

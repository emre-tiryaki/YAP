package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/config"
	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/handler"
)

func main() {
	log.Println("Server is starting...")
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	db, err := database.New(context.Background(), cfg.DBPath)
	if err != nil {
		log.Fatalf("Database error: %v", err)
	}
	defer db.Close()

	log.Println("Database has started!")

	mux := http.NewServeMux()
	mux.HandleFunc("/hello-world", handler.HelloWorld)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on http://localhost%s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}

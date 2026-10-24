package main

import (
	"log"
	"net"
	"net/http"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
)

func main() {
	log.Println("Server is starting...")
	db, err := database.New()
	if err != nil {
		log.Fatalf("Database error: %v", err)
	}
	defer db.Close()
	log.Println("Database has started!")

	mux := http.NewServeMux()
	mux.HandleFunc("/hello-world", database.HelloWorldHandler)

	port := ":8080"

	ln, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Port error: %v", err)
	}
	log.Printf("Server is running on http://localhost%s", port)

	if err = http.Serve(ln, mux); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}

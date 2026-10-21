package main

import (
	"log"
	"net"
	"net/http"

	"github.com/emre-tiryaki/YAP/backend/internal/handlers"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handlers.HelloWorld)

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

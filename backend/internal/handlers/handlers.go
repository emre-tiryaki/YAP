package handlers

import (
	"log"
	"net/http"
)

func HelloWorld(w http.ResponseWriter, r *http.Request) {
	log.Println("The server is working!!!!!")
	w.Write([]byte("Hello world!!"));
}
package handler

import "net/http"

func HelloWorld(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Server is working!!!"))
}
package main

import (
	"log"
	"net/http"
)

func main() {
	log.Println("Initializing server...")
	server := NewServer()

	http.HandleFunc("/ws", server.handleWebSocket)
	go server.run()
	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe error: ", err)
	}
}

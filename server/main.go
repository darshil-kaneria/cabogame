package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// For debugging - should remove in prod
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clients = make(map[*websocket.Conn]bool)
var clientsMutex = sync.Mutex{}

type Message struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	Status  string `json:"status"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	// Register the new client
	clientsMutex.Lock()
	clients[conn] = true
	clientsMutex.Unlock()

	// Unregister the client when the function returns
	defer func() {
		clientsMutex.Lock()
		delete(clients, conn)
		clientsMutex.Unlock()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Println("Error unmarshalling message:", err)
			continue
		}

		fmt.Printf("Received: %+v\n", msg)

		// Broadcast the message to all other clients
		clientsMutex.Lock()
		for client := range clients {
			if client != conn {
				if err := client.WriteJSON(msg); err != nil {
					log.Println(err)
					client.Close()
					delete(clients, client)
				}
			}
		}
		clientsMutex.Unlock()
	}
}

func main() {
	http.HandleFunc("/ws", handler)
	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

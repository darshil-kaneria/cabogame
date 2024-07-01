package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Server struct {
	clients    map[*Client]bool
	games      map[string]*Game
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.Mutex
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewServer() *Server {
	log.Println("Creating new server instance")
	return &Server{
		clients:    make(map[*Client]bool),
		games:      make(map[string]*Game),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func generateGameID() (string, error) {
	bytes := make([]byte, 16)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	gameID := hex.EncodeToString(bytes)

	return gameID, nil
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading to WebSocket:", err)
		return
	}
	log.Println("New WebSocket connection established")
	client := &Client{server: s, conn: conn, send: make(chan []byte, 256)}
	s.register <- client

	go client.writePump()
	go client.readPump()

}

func (s *Server) run() {
	for {
		select {
		case client := <-s.register:
			s.mu.Lock()
			s.clients[client] = true
			s.mu.Unlock()
			log.Println("New client registered")
		case client := <-s.unregister:
			s.mu.Lock()
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				close(client.send)
				log.Println("Client unregistered")
			}
			s.mu.Unlock()
		case message := <-s.broadcast:
			log.Println("Broadcasting message to all clients")
			s.mu.Lock()
			for client := range s.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(s.clients, client)
					log.Println("Client removed due to send buffer full")
				}
			}
			s.mu.Unlock()
		}
	}
}

func (s *Server) broadcastToGame(gameID string, message []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	game, ok := s.games[gameID]
	if !ok {
		log.Printf("Attempted to broadcast to non-existent game: %s", gameID)
		return
	}

	log.Printf("Broadcasting message to game: %s", gameID)
	for client := range game.clients {
		select {
		case client.send <- message:
		default:
			close(client.send)
			delete(game.clients, client)
			log.Printf("Client removed from game %s due to send buffer full", gameID)
		}
	}
}

func (s *Server) createGame(gameID string) *Game {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Creating new game with ID: %s", gameID)
	game := NewGame(gameID)
	s.games[gameID] = game
	return game
}

func (s *Server) joinGame(gameID string, client *Client) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	game, ok := s.games[gameID]
	if !ok {
		log.Printf("Attempted to join non-existent game: %s", gameID)
		return false
	}

	if client.game != nil {
		log.Printf("Client attempted to join game %s while already in a game", gameID)
		return false
	}

	game.addClient(client)
	client.game = game
	log.Printf("Client joined game: %s", gameID)
	return true
}

func (s *Server) leaveGame(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if client.game == nil {
		log.Println("Client attempted to leave game while not in any game")
		return
	}

	log.Printf("Client leaving game: %s", client.game.id)
	client.game.removeClient(client)
	client.game = nil
}

func (s *Server) getTotalPlayersInGames() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	total := 0
	for _, game := range s.games {
		total += len(game.clients)
	}
	log.Printf("Total players in games: %d", total)
	return total
}

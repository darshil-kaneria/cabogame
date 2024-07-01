package main

import (
	"log"
	"sync"
)

type Game struct {
	id      string
	clients map[*Client]bool
	mu      sync.Mutex
}

func NewGame(id string) *Game {
	log.Printf("Creating new game with ID: %s", id)
	return &Game{
		id:      id,
		clients: make(map[*Client]bool),
	}
}

func (g *Game) addClient(client *Client) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.clients[client] = true
	log.Printf("Client added to game %s. Total clients: %d", g.id, len(g.clients))
}

func (g *Game) removeClient(client *Client) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.clients, client)
	log.Printf("Client removed from game %s. Total clients: %d", g.id, len(g.clients))
}

func (g *Game) broadcast(message []byte) {
	g.mu.Lock()
	defer g.mu.Unlock()

	log.Printf("Broadcasting message to all clients in game %s", g.id)
	for client := range g.clients {
		select {
		case client.send <- message:
		default:
			close(client.send)
			delete(g.clients, client)
			log.Printf("Client removed from game %s due to send buffer full", g.id)
		}
	}
}
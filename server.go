package main

import (
	"github.com/gorilla/websocket"
)
// Maintains a set of active clients and broadcasts messages to them
type MatchingServer struct {
	// Registered clients
	clients map[uint16]*websocket.Conn

	// Inbound messages from the clients
	broadcast chan []byte

	//Register requests from the clients
	register chan *Client

	//Unregister requests from clients
	unregister chan *Client
}

func newMatchingServer() *MatchingServer {
	return &MatchingServer{
		clients: make(map[*Client]bool),
		broadcast: make(chan []byte),
		register: make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (ms *MatchingServer) run() {
	for {
		select {
		case client := <-ms.register:
			ms.clients[client]
		case client := <-ms.unregister:
			if _, ok := ms.clients[client]; ok {
				delete(ms.clients, client)
				close(client.send)
			}
		case message := <-ms.broadcast:
			for client := range ms.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(ms.clients, client)
				}
			}
		}
		case message := <-ms.pipe:
			// parse message into a message struct
			// send message text to 
	}
}
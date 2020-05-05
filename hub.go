package main

import (
	"github.com/google/uuid"
)

// SocketMessage stores the message sent over WS and the client who sent it
type SocketMessage struct {
	msg []byte
	sender *Client
}

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[uuid.UUID]*Client

	// Inbound messages from the clients.
	broadcast chan *SocketMessage

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan *SocketMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[uuid.UUID]*Client),
	}
}

func (h *Hub) run(w *World) {
	for {
		select {
		case client := <-h.register:
			h.clients[client.id] = client
			client.send <- generateInitPacket(w)
		case client := <-h.unregister:
			if client := h.clients[client.id]; client.connected {
				delete(h.clients, client.id)
				close(client.send)
			}

			leaveMessage := generateLeavePacket(client.id)
			removeCharacter(w, client.id)

			for id := range h.clients {
				select {
				case h.clients[id].send <- leaveMessage:
				default:
					close(h.clients[id].send)
					delete(h.clients, id)
				}
			}
		case message := <-h.broadcast:
			processMessage(w, message)
			for id := range h.clients {
				select {
				case h.clients[id].send <- message.msg:
				default:
					close(h.clients[id].send)
					delete(h.clients, id)
				}
			}
		}
	}
}

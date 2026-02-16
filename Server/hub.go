// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	//Byffer for new or reconnected clients
	ramBuffer map[int][]byte
	// Registered clients.
	clients map[*Client]bool
	// Inbound messages from the clients.
	broadcast chan []byte
	// Register requests from the clients.
	register chan *Client
	// Unregister requests from clients.
	unregister chan *Client

	Key string
}

func NewHub() *Hub {
	return &Hub{
		ramBuffer:  make(map[int][]byte),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) Run() {
	MsgIndex := 0
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true

			for i := 0; i < len(h.ramBuffer); i++ {
				select {
				case client.send <- h.ramBuffer[i]:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			//Write message to local buffer
			h.ramBuffer[MsgIndex] = message
			MsgIndex++
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

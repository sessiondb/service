// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package service

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all for MVP, restrict in prod
	},
}

type NotificationHub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan interface{}
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.Mutex
}

func NewNotificationHub() *NotificationHub {
	return &NotificationHub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan interface{}),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *NotificationHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Println("WS Client connected")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mu.Unlock()
			log.Println("WS Client disconnected")

		case message := <-h.broadcast:
			msgBytes, err := json.Marshal(message)
			if err != nil {
				log.Printf("WS Error marshaling message: %v", err)
				continue
			}

			h.mu.Lock()
			for client := range h.clients {
				err := client.WriteMessage(websocket.TextMessage, msgBytes)
				if err != nil {
					log.Printf("WS Error writing message: %v", err)
					client.Close()
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *NotificationHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS Error upgrading connection: %v", err)
		return
	}

	h.register <- conn

	// Keep reading for closure or pings
	go func() {
		defer func() {
			h.unregister <- conn
		}()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

func (h *NotificationHub) Broadcast(msg interface{}) {
	h.broadcast <- msg
}

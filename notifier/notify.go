package notifier

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// Upgrader to upgrade HTTP connection to WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client structure to hold the WebSocket connection
type Client struct {
	conn       *websocket.Conn
	send       chan []byte
	clientType string // "dashboard" or "management"
}

// Hub maintains the set of active clients
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.Mutex
}

// Global Hub instance
var Hub_ = Hub{
	clients:    make(map[*Client]bool),
	broadcast:  make(chan []byte),
	register:   make(chan *Client),
	unregister: make(chan *Client),
}

// Run starts the Hub to manage client connections
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			log.Println("New Instant Notification Client Connected")
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Println("Instant Notification client disconnected")
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.Lock()
			log.Printf("Broadcasting to %d Instant Notification clients", len(h.clients))
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					delete(h.clients, client)
					close(client.send)
					log.Println("Removed unresponsive Instant Notification client")
				}
			}
			h.mu.Unlock()
		}
	}
}

// Handle WebSocket connections
func ServeNotifierWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket Upgrade failed:", err)
		return
	}

	client := &Client{conn: conn, send: make(chan []byte, 256)}
	Hub_.register <- client

	go readPump(client)
	go writePump(client)
}

// Read messages from client
func readPump(client *Client) {
	defer func() {
		Hub_.unregister <- client
		client.conn.Close()
	}()

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
		fmt.Println("Received message:", string(message))
	}
}

// Write messages to client
func writePump(client *Client) {
	for message := range client.send {
		if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			break
		}
	}
}

// SendNotifications sends notifications every 5 seconds
func SendNotifications() {
	for {
		time.Sleep(5 * time.Second)
		message := []byte("New notification at " + time.Now().Format(time.RFC3339))
		Hub_.broadcast <- message
	}
}

// Send specific message
func SendNotification(notification NotiferEvent) {
	// Convert the notification struct to JSON
	message, err := json.Marshal(notification)
	if err != nil {
		log.Println("Error marshalling notification:", err)
		return
	}

	log.Println(notification)

	Hub_.broadcast <- message
}

package notifier

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	_ "github.com/microsoft/go-mssqldb"
)

// Upgrader to upgrade HTTP connection to WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client structure to hold the WebSocket connection
type Client struct {
	conn *websocket.Conn
	send chan []byte
}

// Hub maintains the set of active clients
type Hub struct {
	clients             map[*Client]bool
	dashboardClients    map[*Client]bool
	broadcast           chan []byte
	dashboardBroadcast  chan []byte
	register            chan *Client
	unregister          chan *Client
	dashboardRegister   chan *Client
	dashboardUnregister chan *Client
	mu                  sync.Mutex
}

type MonitorMetaData struct {
	DownTime             string     `json:"DownTime"`
	AcknowledgedDateTime *time.Time `json:"AcknowledgedDateTime,omitempty"`
	SnoozeUntil          *time.Time `json:"SnoozeUntil,omitempty"`
	LastServiceUptime    *time.Time `json:"LastServiceUptime,omitempty"`
	LastCheckTime        *time.Time `json:"LastCheckTime,omitempty"`
	CreatedAt            time.Time  `json:"CreatedAt"`
}

type ServiceMonitorData struct {
	// SystemMonitorId uuid.UUID `json:""`
	SystemMonitorId string `json:"SystemMonitorId"`
	Name            string
	Host            string `json:"IPAddress"`
	Port            int
	VP              bool `json:"IsMonitored"` // Is monitoring active?
	// Status               sql.NullString
	// LastServiceUptime    sql.NullTime
	// LastCheckTime        sql.NullTime
	// IsAcknowledged       bool // Is failing service acknowledged?
	// Device               ServiceType
	// LiveCheckFlag        int
	// DownTime             string
	// AcknowledgedDateTime sql.NullTime
	// SnoozeUntil          sql.NullTime `json:"SnoozeUntil"`
	// CreatedAt            sql.NullTime

	Status         *string `json:"HealthStatus,omitempty"`
	IsAcknowledged bool    `json:"IsServiceIssueAcknowledged"`
	Device         string  `json:"Device"`
	LiveCheckFlag  int     `json:"LiveCheckFlag"`

	Metadata MonitorMetaData `json:"Metadata"`
	// AgentAPIBaseURL   string                 `json:"agent_api"`
	// AgentRepository internal.AgentRepository
}

type DeviceGroup struct {
	Title string               `json:"Title"`
	Data  []ServiceMonitorData `json:"Data"`
}

// Global Hub instance
var Hub_ = Hub{
	clients:             make(map[*Client]bool),
	dashboardClients:    make(map[*Client]bool),
	broadcast:           make(chan []byte),
	dashboardBroadcast:  make(chan []byte),
	register:            make(chan *Client),
	unregister:          make(chan *Client),
	dashboardRegister:   make(chan *Client),
	dashboardUnregister: make(chan *Client),
}

func fetchData(db *sql.DB) ([]DeviceGroup, error) {
	// var ctx context.Context
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	rows, err := db.Query("EXEC ServiceReport @SERVICE_LEVEL = 'ALL', @VP = 1;")
	if err != nil {
		return nil, fmt.Errorf("error querying services: %v", err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			// return nil, err
		}
	}(rows)

	deviceMap := make(map[string][]ServiceMonitorData)

	for rows.Next() {
		var service ServiceMonitorData
		var uuidBytes []byte // Changed from string to []byte to handle binary UUID

		err := rows.Scan(
			&uuidBytes, // Now expecting binary data
			// &uuidStr,
			&service.Name,
			&service.Host,
			&service.Port,
			&service.VP,
			&service.Status,
			&service.Metadata.LastServiceUptime,
			&service.Metadata.LastCheckTime,
			&service.Device,
			&service.LiveCheckFlag,
			&service.Metadata.DownTime,
			&service.IsAcknowledged,
			&service.Metadata.AcknowledgedDateTime,
			&service.Metadata.SnoozeUntil,
			&service.Metadata.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("error scanning service row: %v", err)
		}

		// service.SystemMonitorId, err = uuid.Parse(uuidStr)
		// if err != nil {
		// 	slog.Info(uuidStr)
		// 	slog.Error(err.Error())
		// }

		// Convert binary UUID to proper string format if needed
		// This assumes the UUID is stored as 16 bytes in the database
		if len(uuidBytes) == 16 {
			uuidStr, err := convertBinaryUUIDToString(uuidBytes)
			if err != nil {
				// Log the error but continue processing
				log.Printf("Error converting UUID: %v", err)
			} else {
				service.SystemMonitorId = uuidStr
			}
		}

		// service.SystemMonitorId, err = uuid.Validate(uuidStr)
		// if err != nil {
		// 	slog.Error(err.Error())
		// }

		// Group by device
		deviceMap[service.Device] = append(deviceMap[service.Device], service)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error after row iteration: %v", err)
	}

	// Convert the map to the desired slice of DeviceGroup
	var result []DeviceGroup
	for device, services := range deviceMap {
		result = append(result, DeviceGroup{
			Title: device,
			Data:  services,
		})
	}

	return result, nil
}

// Helper function to convert 16-byte binary UUID to string representation
func convertBinaryUUIDToString(b []byte) (string, error) {
	if len(b) != 16 {
		return "", fmt.Errorf("invalid UUID length: %d", len(b))
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// Run starts the Hub to manage client connections
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			log.Println("New Client Connected")
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.dashboardRegister:
			h.mu.Lock()
			log.Println("New Dashboard Client Connected")
			h.dashboardClients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case client := <-h.dashboardUnregister:
			h.mu.Lock()
			if _, ok := h.dashboardClients[client]; ok {
				delete(h.dashboardClients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					delete(h.clients, client)
					close(client.send)
				}
			}
			h.mu.Unlock()

		case message := <-h.dashboardBroadcast:
			h.mu.Lock()
			for client := range h.dashboardClients {
				select {
				case client.send <- message:
				default:
					delete(h.dashboardClients, client)
					close(client.send)
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

// Broadcast data to all clients
func BroadcastDashboardData(db *sql.DB) {
	for {
		data, err := fetchData(db)
		if err != nil {
			log.Println("Error fetching data:", err)
			time.Sleep(5 * time.Second) // Retry delay
			continue
		}

		// Serialize to JSON (which is a []byte)
		jsonData, err := json.Marshal(data)
		if err != nil {
			fmt.Println("Error marshaling data:", err)
			return
		}

		Hub_.dashboardBroadcast <- jsonData
		time.Sleep(30 * time.Second) // Adjust update interval
	}
}

// Handle new WebSocket connections
func ServeSyntheticDashboard(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading to WebSocket:", err)
		return
	}

	client := &Client{conn: conn, send: make(chan []byte, 256)}
	Hub_.dashboardRegister <- client

	go readDashboardPump(client)
	go writeDashboardPump(client)
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

// Read Dashboard messages from client
func readDashboardPump(client *Client) {
	defer func() {
		Hub_.dashboardUnregister <- client
		client.conn.Close()
	}()

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			break
		}
		fmt.Println("Received message:", string(message))
	}
}

// Write messages to client
func writeDashboardPump(client *Client) {
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

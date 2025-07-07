package notifier

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/ZEGIFTED/MS.GoMonitor/pkg/utils"
	"github.com/gorilla/websocket"
	"github.com/lib/pq" // PostgreSQL driver
)

// Upgrader to upgrade HTTP connection to WebSocket
var upgrader_ = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// DashboardClient structure to hold the WebSocket connection
type DashboardClient struct {
	conn       *websocket.Conn
	send       chan []byte
	clientType string // "dashboard" or "management"
}

// DashboardHub maintains the set of active clients
type DashboardHub struct {
	dashboardClients map[*DashboardClient]bool
	broadcast        chan BroadcastMessage
	register         chan *DashboardClient
	unregister       chan *DashboardClient

	deviceGroups      map[string]DeviceGroup
	serviceMonitors   map[string]ServiceMonitorData
	lastDashboardData []byte
	mu                sync.Mutex
}

var DashHub = DashboardHub{
	dashboardClients: make(map[*DashboardClient]bool),
	broadcast:        make(chan BroadcastMessage),
	register:         make(chan *DashboardClient),
	unregister:       make(chan *DashboardClient),

	deviceGroups:    make(map[string]DeviceGroup),
	serviceMonitors: make(map[string]ServiceMonitorData),
}

// Run starts the Hub to manage client connections
func (h *DashboardHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			slog.Info(fmt.Sprintf("New %s Client Connected", client.clientType))
			h.dashboardClients[client] = true
			h.mu.Unlock()

			// Send initial data based on client type
			switch client.clientType {
			case "dashboard":
				if h.lastDashboardData != nil {
					select {
					case client.send <- h.lastDashboardData:
						log.Println("Sent initial dashboard data to new client")
					default:
						log.Println("Failed to send initial dashboard data - channel blocked")
					}
				}
			case "management":
				h.sendInitialManagementData(client)
			}

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.dashboardClients[client]; ok {
				delete(h.dashboardClients, client)
				close(client.send)
				slog.Info(fmt.Sprintf("%s Client Disconnected", client.clientType))
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.Lock()
			targetCount := 0
			for client := range h.dashboardClients {
				if message.Target == "all" ||
					(message.Target == "dashboard" && client.clientType == "dashboard") ||
					(message.Target == "management" && client.clientType == "management") {
					targetCount++

					select {
					case client.send <- message.Data:
						if message.Target == "dashboard" || message.Target == "all" {
							h.lastDashboardData = message.Data
						}
					default:
						close(client.send)
						delete(h.dashboardClients, client)
						log.Printf("Removed unresponsive %s client", client.clientType)
					}
				}
			}
			log.Printf("Broadcasted %s message to %d clients", message.Target, targetCount)
			h.mu.Unlock()
		}
	}
}

func (h *DashboardHub) sendInitialManagementData(client *DashboardClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Send all groups
	groups := make([]DeviceGroup, 0, len(h.deviceGroups))
	for _, group := range h.deviceGroups {
		groups = append(groups, group)
	}
	groupsData, _ := json.Marshal(map[string]interface{}{
		"type": "initialGroups",
		"data": groups,
	})

	select {
	case client.send <- groupsData:
		log.Println("Sent initial groups data to management client")
	default:
		log.Println("Failed to send initial groups data - channel blocked")
	}

	// Send all devices
	devices := make([]ServiceMonitorData, 0, len(h.serviceMonitors))
	for _, device := range h.serviceMonitors {
		devices = append(devices, device)
	}
	devicesData, _ := json.Marshal(map[string]interface{}{
		"type": "initialDevices",
		"data": devices,
	})

	select {
	case client.send <- devicesData:
		log.Println("Sent initial devices data to management client")
	default:
		log.Println("Failed to send initial devices data - channel blocked")
	}
}

// Broadcast data to all clients
func BroadcastDashboardData(db *sql.DB) {
	for {
		data, err := fetchData(db)
		if err != nil {
			log.Println("Error fetching dashboard data:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		jsonData, err := json.Marshal(data)
		if err != nil {
			log.Println("Error marshaling dashboard data:", err)
			continue
		}

		DashHub.broadcast <- BroadcastMessage{
			Target: "dashboard",
			Data:   jsonData,
		}
		time.Sleep(30 * time.Second)
	}
}

// func fetchData(db *sql.DB) ([]DeviceGroup, error) {
// 	if db == nil {
// 		return nil, fmt.Errorf("database connection is nil")
// 	}

// 	rows, err := db.Query("EXEC ServiceReport @SERVICE_LEVEL = 'ALL', @IsMonitored = 1;")
// 	if err != nil {
// 		return nil, fmt.Errorf("error querying services: %v", err)
// 	}
// 	defer rows.Close()

// 	deviceMap := make(map[string][]ServiceMonitorData)

// 	for rows.Next() {
// 		var service ServiceMonitorData
// 		var uuidBytes []byte

// 		err := rows.Scan(
// 			&uuidBytes,
// 			&service.Name,
// 			&service.IPAddress,
// 			&service.Port,
// 			&service.IsMonitored,
// 			&service.Status,
// 			&service.Metadata.LastServiceUptime,
// 			&service.Metadata.LastCheckTime,
// 			&service.Device,
// 			&service.LiveCheckFlag,
// 			&service.Metadata.DownTime,
// 			&service.IsAcknowledged,
// 			&service.Metadata.AcknowledgedDateTime,
// 			&service.Metadata.SnoozeUntil,
// 			&service.Metadata.CreatedAt,
// 		)

// 		if err != nil {
// 			return nil, fmt.Errorf("error scanning service row: %v", err)
// 		}

// 		if len(uuidBytes) == 16 {
// 			uuidStr, err := convertBinaryUUIDToString(uuidBytes)
// 			if err != nil {
// 				log.Printf("Error converting UUID: %v", err)
// 			} else {
// 				service.SystemMonitorId = uuidStr
// 			}
// 		}

// 		deviceMap[service.Device] = append(deviceMap[service.Device], service)
// 	}

// 	if err := rows.Err(); err != nil {
// 		return nil, fmt.Errorf("error after row iteration: %v", err)
// 	}

// 	var result []DeviceGroup
// 	for device, services := range deviceMap {
// 		deviceIDs := make([]string, len(services))
// 		for i, svc := range services {
// 			deviceIDs[i] = svc.SystemMonitorId
// 		}

// 		result = append(result, DeviceGroup{
// 			Title:     device,
// 			Devices:   services,
// 			DeviceIDs: deviceIDs,
// 		})
// 	}

// 	return result, nil
// }

func fetchData(db *sql.DB) ([]ServiceMonitorData, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	// rows, err := db.Query("EXEC ServiceReport @SERVICE_LEVEL = 'ALL', @IsMonitored = 1;")
	query := `SELECT "SystemMonitorId",
       "ServiceName",
       "IPAddress",
       "Port",
       "IsMonitored",
	   "CurrentHealthCheck",
       "LastServiceUpTime",
       "LastCheckTime",
       "Device",
       "FailureCount",
       "RetryCount",
       "IsAcknowledged",
       "SnoozeUntil",
       "CheckInterval",
       "Plugins"
FROM servicereport('ALL', NULL, TRUE, NULL);`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying services: %v", err)
	}
	defer rows.Close()

	// deviceMap := make(map[string][]ServiceMonitorData)
	var services []ServiceMonitorData
	var plugins pq.StringArray

	for rows.Next() {
		var service ServiceMonitorData

		err := rows.Scan(
			&service.SystemMonitorId,
			&service.Name,
			&service.IPAddress,
			&service.Port,
			&service.IsMonitored,
			&service.CurrentHealthCheck,
			&service.Metadata.LastServiceUptime,
			&service.Metadata.LastCheckTime,
			&service.Device,
			&service.FailureCount,
			&service.RetryCount,
			&service.IsAcknowledged,
			&service.Metadata.SnoozeUntil,
			&service.CheckInterval,
			&plugins,
		)

		if err != nil {
			return nil, fmt.Errorf("error scanning service row: %v", err)
		}

		if service.Metadata.LastServiceUptime.Valid {
			service.Metadata.DownTime = utils.GetServiceDownTime(service.Metadata.LastServiceUptime.Time)
		} else {
			service.Metadata.DownTime = "_D _M _S" // or any fallback value
		}

		service.Plugins = plugins
		// service.StatusInfo = constants.GetStatusInfo(service.LiveCheckFlag, "")

		services = append(services, service)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error after row iteration: %v", err)
	}

	return services, nil
}

// Helper function to convert 16-byte binary UUID to string representation
func convertBinaryUUIDToString(b []byte) (string, error) {
	if len(b) != 16 {
		return "", fmt.Errorf("invalid UUID length: %d", len(b))
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// Handle new WebSocket connections
func ServeSyntheticDashboard(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader_.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading to WebSocket:", err)
		return
	}

	client := &DashboardClient{conn: conn, send: make(chan []byte, 256), clientType: "dashboard"}
	DashHub.register <- client

	go readDashboardPump(client)
	go writeDashboardPump(client)
}

func ServeManagementInterface(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader_.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading to WebSocket:", err)
		return
	}

	client := &DashboardClient{conn: conn, send: make(chan []byte, 256), clientType: "management"}
	DashHub.register <- client

	go readDashboardPump(client)
	go writeDashboardPump(client)
}

func handleManagementMessage(message []byte) {
	var msg struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Error parsing management message: %v", err)
		return
	}

	DashHub.mu.Lock()
	defer DashHub.mu.Unlock()

	switch msg.Type {
	case "createGroup":
		var group DeviceGroup
		if err := json.Unmarshal(msg.Data, &group); err != nil {
			log.Printf("Error unmarshaling createGroup data: %v", err)
			return
		}
		group.ID = fmt.Sprintf("group-%d", time.Now().UnixNano())
		group.CreatedAt = time.Now()
		group.UpdatedAt = time.Now()
		DashHub.deviceGroups[group.ID] = group
		broadcastGroupUpdate("groupCreated", group)

	case "updateGroup":
		var group DeviceGroup
		if err := json.Unmarshal(msg.Data, &group); err != nil {
			log.Printf("Error unmarshaling updateGroup data: %v", err)
			return
		}
		if _, exists := DashHub.deviceGroups[group.ID]; exists {
			group.UpdatedAt = time.Now()
			DashHub.deviceGroups[group.ID] = group
			broadcastGroupUpdate("groupUpdated", group)
		}

	case "deleteGroup":
		var groupID string
		if err := json.Unmarshal(msg.Data, &groupID); err != nil {
			log.Printf("Error unmarshaling deleteGroup data: %v", err)
			return
		}
		if _, exists := DashHub.deviceGroups[groupID]; exists {
			delete(DashHub.deviceGroups, groupID)
			broadcastGroupUpdate("groupDeleted", groupID)
		}

	case "updateDevice":
		var device ServiceMonitorData
		if err := json.Unmarshal(msg.Data, &device); err != nil {
			log.Printf("Error unmarshaling updateDevice data: %v", err)
			return
		}
		DashHub.serviceMonitors[device.SystemMonitorId] = device
		broadcastDeviceUpdate(device)
	}
}

func broadcastGroupUpdate(action string, data interface{}) {
	message := map[string]any{
		"type": action,
		"data": data,
	}

	jsonData, _ := json.Marshal(message)
	DashHub.broadcast <- BroadcastMessage{
		Target: "management",
		Data:   jsonData,
	}
}

// broadcastDeviceUpdate
func broadcastDeviceUpdate(device ServiceMonitorData) {
	message := map[string]any{
		"type": "deviceUpdate",
		"data": device,
	}
	jsonData, _ := json.Marshal(message)
	DashHub.broadcast <- BroadcastMessage{Target: "all", Data: jsonData}
}

// Read Dashboard messages from client
func readDashboardPump(client *DashboardClient) {
	defer func() {
		DashHub.unregister <- client
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
		// fmt.Println("Received message:", string(message))

		if client.clientType == "management" {
			handleManagementMessage(message)
		}
	}
}

// Write messages to client
func writeDashboardPump(client *DashboardClient) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-client.send:
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

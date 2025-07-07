package notifier

import (
	"database/sql"
	"time"

	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
)

type NotiferEvent struct {
	Title      string
	Identifier string
	Timestamp  string
	Message    string
}

type BroadcastMessage struct {
	Target  string // "all", "dashboard", "management"
	Data    []byte
	GroupID string // Optional for group-specific messages
}
type MonitorMetaData struct {
	DownTime             string       `json:"DownTime"`
	AcknowledgedDateTime *time.Time   `json:"AcknowledgedDateTime,omitempty"`
	SnoozeUntil          sql.NullTime `json:"SnoozeUntil,omitempty"`
	LastServiceUptime    sql.NullTime `json:"LastServiceUptime,omitempty"`
	LastCheckTime        sql.NullTime `json:"LastCheckTime,omitempty"`
	CreatedAt            time.Time    `json:"CreatedAt"`
}

type ServiceMonitorData struct {
	// SystemMonitorId uuid.UUID `json:""`
	SystemMonitorId string `json:"SystemMonitorId"`
	Name            string
	IPAddress       string `json:"IPAddress"`
	Port            int
	IsMonitored     bool `json:"IsMonitored"`  // Is monitoring active?
	FailureCount    int  `json:"failureCount"` // Number of consecutive failures
	RetryCount      int  `json:"retryCount"`   // Number of retry attempts
	// IsAcknowledged       bool // Is failing service acknowledged?
	// Device               ServiceType
	// LiveCheckFlag        int
	// DownTime             string
	// AcknowledgedDateTime sql.NullTime
	// SnoozeUntil          sql.NullTime `json:"SnoozeUntil"`
	// CreatedAt            sql.NullTime
	StatusInfo         constants.StatusInfo `json:"HealthStatusInfo,omitempty"`
	IsAcknowledged     bool                 `json:"IsServiceIssueAcknowledged"`
	Device             string               `json:"Device"`
	CurrentHealthCheck string               `json:"LiveCheckFlag"`
	CheckInterval      string               `json:"checkInterval"` // Cron-style interval
	Plugins            []string             `json:"Plugins"`       // This will hold the parsed plugins
	Metadata           MonitorMetaData      `json:"Metadata"`
	// AgentAPIBaseURL   string                 `json:"agent_api"`
	// AgentRepository internal.AgentRepository
}

type DeviceGroup struct {
	ID          string               `json:"id"`
	Title       string               `json:"title"`
	Description string               `json:"description,omitempty"`
	Devices     []ServiceMonitorData `json:"devices,omitempty"`
	DeviceIDs   []string             `json:"deviceIds"`
	CreatedAt   time.Time            `json:"createdAt"`
	UpdatedAt   time.Time            `json:"updatedAt"`
}

package internal

import (
	"time"

	"github.com/google/uuid"
)

// ProcessResourceUsage represents a single process entry returned by the Agent API endpoint.
type ProcessResourceUsage struct {
	Username      string  `json:"username"`
	PID           int     `json:"pid"`
	CPUPercent    float64 `json:"cpu_percent"`
	CreateTime    float64 `json:"create_time"`
	Status        string  `json:"status"`
	MemoryPercent float64 `json:"memory_percent"`
	Name          string  `json:"name"`
}

type ProcessResponse []ProcessResourceUsage

type NotificationRecipient struct {
	SystemMonitorId uuid.UUID `json:"systemMonitorId"`
	ServiceName     string    `json:"serviceName"`
	UserName        string
	Email           string
	PhoneNumber     string
	SlackId         string
	GroupName       string
	Platform        string
}

type NotificationRecipients struct {
	Users []NotificationRecipient
	//Groups      []Group
}

type User struct {
	Email string
	Name  string
}

type ServiceAlertEvent struct {
	ServiceName     string
	SystemMonitorId uuid.UUID `json:"systemMonitorId"`
	Message         string
	Device          string
	Severity        string
	Timestamp       time.Time
	AgentRepository AgentRepository
	ServiceStats    ProcessResponse
	AgentAPI        string
}

type Metrics struct {
	CPU     bool `json:"CPU"`
	Disk    bool `json:"Disk"`
	Memory  bool `json:"Memory"`
	Latency bool `json:"Latency"`
}

type MetricsThreshold struct {
	CPU struct {
		High int `json:"high"`
		Low  int `json:"low"`
		Mid  int `json:"mid"`
	} `json:"CPU"`
	Disk struct {
		High int `json:"high"`
		Low  int `json:"low"`
		Mid  int `json:"mid"`
	} `json:"Disk"`
	Latency struct {
		High int `json:"high"`
		Low  int `json:"low"`
		Mid  int `json:"mid"`
	} `json:"Latency"`
}

type Config struct {
	AgentID            string           `json:"AgentID"`
	AgentSleepInterval int              `json:"AgentSleepInterval"`
	ProvisionedState   string           `json:"ProvisionedState"`
	LicenseKey         string           `json:"LicenseKey"`
	APIBaseUrl         string           `json:"APIBaseUrl"`
	Metrics            Metrics          `json:"Metrics"`
	MetricsThreshold   MetricsThreshold `json:"MetricsThreshold"`
	AgentMonitoredLogs []interface{}    `json:"AgentMonitoredLogs"` // Use interface{} to handle null or any type
}

type AgentThresholdResponse struct {
	Config Config `json:"config"`
}

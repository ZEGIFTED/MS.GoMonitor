package internal

import (
	"time"

	"github.com/google/uuid"
)

// ProcessResourceUsage represents a single process entry returned by the Agent API endpoint.
type ProcessResourceUsage struct {
	PID           int     `json:"pid"`
	MemoryPercent float64 `json:"memory_percent"`
	CPUPercent    float64 `json:"cpu_percent"`
	Status        string  `json:"status"`
	Name          string  `json:"name"`
	CreateTime    float64 `json:"create_time"`
	Username      string  `json:"username"`
}

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
	Severity        string
	Timestamp       time.Time
	AgentRepository AgentRepository
	AgentAPI        string
}

type AgentThresholdResponse struct {
	MetricsThreshold map[string]interface{} `json:"metricThreshold"`
	Metrics          map[string]bool        `json:"metrics"`
}

package internal

import (
	"github.com/google/uuid"
	"time"
)

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

type ServiceAlertEvent struct {
	ServiceName     string
	SystemMonitorId uuid.UUID `json:"systemMonitorId"`
	Message         string
	Severity        string
	Timestamp       time.Time
}

type AgentThresholdResponse struct {
	MetricsThreshold map[string]interface{} `json:"metricThreshold"`
	Metrics          map[string]bool        `json:"metrics"`
}

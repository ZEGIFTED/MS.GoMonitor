package monitors

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/ZEGIFTED/MS.GoMonitor/internal"
	"github.com/ZEGIFTED/MS.GoMonitor/internal/repository"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/messaging"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

type ServiceType string

const (
	ServiceMonitorAgent      ServiceType = "AGENT"
	ServiceMonitorWebModules ServiceType = "Web Modules"
	ServiceMonitorSNMP       ServiceType = "Network"
	ServiceMonitorServer     ServiceType = "Server"
)

type AgentServiceChecker struct{}
type WebModulesServiceChecker struct{}
type SNMPServiceChecker struct{}
type ServerHealthChecker struct{}

type ServiceMonitorData struct {
	SystemMonitorId uuid.UUID `json:"system_monitor_id"`
	Name            string
	Host            string
	Port            int
	VP              bool // Is monitoring active?
	IsAcknowledged  bool // Is failing service acknowledged?
	Device          ServiceType
	FailureCount    int
	RetryCount      int
	Configuration   map[string]interface{} // Custom Settings for this service
	CheckInterval   string                 `json:"check_interval"`
	SnoozeUntil     sql.NullTime           `json:"snooze_until"`
	AgentAPIBaseURL string                 `json:"agent_api"`
	AgentRepository repository.AgentRepository
}

// ServiceMonitorStatus represents the current status of a monitored service
type ServiceMonitorStatus struct {
	Name              string      `json:"name"`
	Device            ServiceType `json:"device"`
	LiveCheckFlag     int
	Status            string    `json:"status"`
	LastCheckTime     time.Time `json:"last_checked"`
	LastServiceUpTime time.Time `json:"last_service_up_time"`
	FailureCount      int       `json:"failure_count"`
	//LastFailure       time.Time `json:"last_failure"`
	//LastErrorLog string `json:"last_error_log"`
}

type ServiceChecker interface {
	Check(config ServiceMonitorData, ctx context.Context, Db *sql.DB) (ServiceMonitorStatus, bool)
}

type ServiceMonitor struct {
	Db       *sql.DB              // Database connection
	Services []ServiceMonitorData // List of services to monitor
	//StatusTracking map[string]*ServiceMonitorStatus // Current status of each service
	StatusTracking sync.Map     // Concurrency-safe map for service statuses
	MU             sync.RWMutex // For thread safety

	Checkers            map[ServiceType]ServiceChecker // Different types of checks
	NotificationHandler *messaging.NotificationManager
	Cron                *cron.Cron
	Ctx                 context.Context
	Cancel              context.CancelFunc

	AlertCache sync.Map                        // Stores last alert timestamps per service
	Alerts     chan internal.ServiceAlertEvent // Buffered channel for processing alerts
}

package monitors

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/ZEGIFTED/MS.GoMonitor/internal"
	"github.com/ZEGIFTED/MS.GoMonitor/internal/repository"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/messaging"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

type ServiceType string
type Engines string

const (
	ServiceMonitorAgent      ServiceType = "AGENT"
	ServiceMonitorWebModules ServiceType = "Web Modules"
	ServiceMonitorSNMP       ServiceType = "Network"
	ServiceMonitorServer     ServiceType = "Server"
	DockerEngine             Engines     = "Docker"
)

type ServiceMonitorData struct {
	SystemMonitorId    uuid.UUID `json:"systemMonitorId"`
	Name               string
	Host               string
	Port               int
	IsMonitored        bool `json:"IsMonitored"` // Is monitoring active?
	CurrentHealthCheck string
	IsAcknowledged     bool           `json:"isAcknowledged"` // Is failing service acknowledged?
	Device             ServiceType    `json:"device"`         // Type of service being monitored
	Engine             Engines        `json:"engine"`         // Engine/technology used
	FailureCount       int            `json:"failureCount"`   // Number of consecutive failures
	RetryCount         int            `json:"retryCount"`     // Number of retry attempts
	Configuration      map[string]any `json:"configuration"`  // Custom Settings for this service
	CheckInterval      string         `json:"checkInterval"`  // Cron-style interval
	SnoozeUntil        sql.NullTime   `json:"snoozeUntil"`    // Time until alerts are snoozed

	AgentAPIBaseURL string `json:"agentApiBaseUrl"` // Base URL for agent API
	AgentRepository repository.AgentRepository

	Plugins []string `json:"Plugins"` // This will hold the parsed plugins
}

// ServiceMonitorStatus represents the current status of a monitored service
type ServiceMonitorStatus struct {
	SystemMonitorId uuid.UUID `json:"systemMonitorId"`
	// PluginResults     map[string]PluginResult `json:"pluginResults"`
}

// PluginMonitoringResult represents plugin-specific monitoring result
type MonitoringResult struct {
	ID                string               `json:"id"`
	Name              string               `json:"name,omitempty"`
	Device            ServiceType          `json:"device,omitempty"`
	SystemMonitorId   string               `json:"SystemMonitorId"`
	ServicePluginID   string               `json:"service_plugin_id"`
	HealthReport      constants.StatusInfo `json:"HealthReport,omitempty"`
	Details           map[string]any
	LastCheckTime     time.Time `json:"last_checked"`
	LastServiceUpTime time.Time `json:"last_service_up_time"`
	FailureCount      int       `json:"failure_count"`
	// Data            *string   `json:"data,omitempty"`
	// ExecutionTime   *int64    `json:"execution_time,omitempty"`
	// ErrorDetails    *string   `json:"error_details,omitempty"`
}

type MonitoringBatch struct {
	MainResult    ServiceMonitorStatus
	PluginResults []MonitoringResult `json:"pluginMonitoringResult"`
}

// type ServiceChecker interface {
// 	Check(config ServiceMonitorData, ctx context.Context, Db *sql.DB) (ServiceMonitorStatus, bool)
// }

// type AgentServiceChecker struct{}
// type WebModulesServiceChecker struct{}
// type SNMPServiceChecker struct{}
// type ServerHealthChecker struct{}

type ServiceMonitorPlugin interface {
	// Name returns the plugin's name
	Name() string

	// Description returns a human-readable description
	Description() string

	// SupportedTypes returns the service types this plugin can monitor
	SupportedTypes() []ServiceType

	// Initialize the plugin with configuration
	Initialize(map[string]any) error

	// Performs the monitoring check
	Check(context.Context, *sql.DB, ServiceMonitorData) (MonitoringResult, error)

	Cleanup() error
}

type MonitoringEngine struct {
	Db             *sql.DB              // Database connection
	Services       []ServiceMonitorData // List of services to monitor
	StatusTracking sync.Map             // Concurrency-safe status map of each service
	MU             sync.RWMutex         // For thread safety

	DefaultHealth ServiceMonitorPlugin
	// Checkers          map[ServiceType]ServiceChecker
	Plugins           map[string]ServiceMonitorPlugin
	PluginInitialized map[string]bool
	// ServicePlugins      map[string]ServiceMonitorPlugin
	NotificationHandler *messaging.NotificationManager
	Cron                *cron.Cron
	Ctx                 context.Context
	Cancel              context.CancelFunc
	AlertCache          sync.Map
	Alerts              chan internal.ServiceAlertEvent // Buffered channel for processing alerts
}

package monitors

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/ZEGIFTED/MS.GoMonitor/internal"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/messaging"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

type ServiceType string

const (
	ServiceMonitorAgent      ServiceType = "AGENT"
	ServiceMonitorWebModules ServiceType = "Web Modules"
	ServiceMonitorSNMP_V2    ServiceType = "NetworkV2"
	ServiceMonitorSNMP_V3    ServiceType = "NetworkV3"
	ServiceMonitorServer     ServiceType = "Server"
)

type AgentServiceChecker struct{}
type WebModulesServiceChecker struct{}
type SNMPServiceCheckerV2 struct{}
type SNMPServiceCheckerV3 struct{}
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
	Configuration   map[string]interface{} // Settings for this service
	CheckInterval   string                 `json:"check_interval"`
	SnoozeUntil     sql.NullTime           `json:"snooze_until"`
	AgentAPIBaseURL string                 `json:"agent_api"`
	AgentRepository internal.AgentRepository
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

type ServiceChecker interface {
	Check(config ServiceMonitorData, ctx context.Context, Db *sql.DB) (ServiceMonitorStatus, bool)
}

// AgentInfo represents the complete metrics information for an agent
type AgentInfo struct {
	Version   string `json:"version"`
	AgentID   string `json:"agent_id"`
	IPAddress string `json:"IPAddress"`
	Name      string `json:"name"`
	OS        string `json:"os"`
	//LastSync   sql.NullTime `json:"lastSync"`
	SDKVersion string `json:"SDKVersion"`
	Metrics    []Metric
	Disks      []DiskMetric
}

// AgentMetricResponse represents the complete metrics data for an agent
type AgentMetricResponse struct {
	Status     string     `json:"status"`
	SystemInfo SystemInfo `json:"systemInfo"`
	Uptime     string     `json:"uptime"`
	AgentInfo  AgentInfo  `json:"agent_info"`
}

type DiskMetric struct {
	Drive      string `json:"drive"`
	Size       int64  `json:"size"`
	Free       int64  `json:"free"`
	Used       int64  `json:"used"`
	FormatSize string `json:"formatSize"`
	FormatFree string `json:"formatFree"`
}

// SystemInfo represents system metrics (CPU, memory, disk)
type SystemInfo struct {
	CPU    [][]float64  `json:"cpu"`    // Each entry is [timestamp, usage]
	Memory [][]float64  `json:"memory"` // Each entry is [timestamp, usage]
	Disk   []DiskMetric `json:"disk"`
}

type Metric struct {
	Timestamp    int64
	TimestampMem int64
	CPUUsage     float64
	MemoryUsage  float64
	AgentID      string
}

type AgentEscalation struct {
	AppId   uuid.UUID `json:"appId"`
	AgentId string    `json:"agentId"`
	Metric  string    `json:"metric"`
	Cause   string    `json:"cause"`
}

// ServerResource represents the server resource utilization data
type ServerResource struct {
	ServerName        string
	CPUUtilization    float64
	MemoryUtilization float64
	DiskUtilization   float64
}

// NetworkDevice represents the network device bandwidth utilization data
type NetworkDeviceMetric struct {
	DeviceName           string
	Uptime               string
	Interfaces           string
	BandwidthUtilization float64
}

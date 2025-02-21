package monitors

import (
	"context"
	"database/sql"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"log"
	"sync"
	"time"
)

type ServiceMonitorConfig struct {
	Id             uuid.UUID `json:"AppID"`
	Name           string
	Host           string
	Port           int
	VP             bool // Is monitoring active?
	IsAcknowledged bool // Is failing service monitoring acknowledged?
	Device         ServiceType
	RetryCount     int
	Configuration  map[string]interface{} // Settings for this service
	CheckInterval  string                 `json:"check_interval"`
	HealthCheckURL string                 `json:"health_check_url"`
	SnoozeUntil    sql.NullTime           `json:"snooze_until"`
}

type ServiceType string

const (
	ServiceMonitorAgent      ServiceType = "AGENT"
	ServiceMonitorWebModules ServiceType = "Web Modules"
	ServiceMonitorSNMP       ServiceType = "Network"
)

// ServiceMonitorStatus represents the current status of a monitored service
type ServiceMonitorStatus struct {
	Name              string      `json:"name"`
	Device            ServiceType `json:"device"`
	LiveCheckFlag     int
	Status            string    `json:"status"`
	LastCheckTime     time.Time `json:"last_checked"`
	LastServiceUpTime time.Time `json:"last_service_up_time"`
	FailureCount      int       `json:"failure_count"`
	LastErrorLog      string    `json:"last_error_log"`
}

type ServiceMonitor struct {
	Db             *sql.DB                          // Database connection
	Services       []ServiceMonitorConfig           // List of services to monitor
	StatusTracking map[string]*ServiceMonitorStatus // Current status of each service
	MU             sync.RWMutex                     // For thread safety
	Logger         *log.Logger                      // For logging
	Checkers       map[ServiceType]ServiceChecker   // Different types of checks
	Cron           *cron.Cron
	Ctx            context.Context
	Cancel         context.CancelFunc
}

type ServiceChecker interface {
	Check(config ServiceMonitorConfig) (bool, ServiceMonitorStatus)
}

// AgentInfo represents the complete metrics information for an agent
type AgentInfo struct {
	Version    string       `json:"version"`
	AgentID    string       `json:"agent_id"`
	IPAddress  string       `json:"IPAddress"`
	Name       string       `json:"name"`
	OS         string       `json:"os"`
	LastSync   sql.NullTime `json:"lastSync"`
	SDKVersion string       `json:"SDKVersion"`
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
type NetworkDevice struct {
	DeviceName           string
	BandwidthUtilization float64
}

// FastAPIResponse represents the response from the FastAPI agent
type FastAPIResponse struct {
	Context string `json:"context"`
}

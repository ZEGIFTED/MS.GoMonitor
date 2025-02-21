package utils

import (
	"context"
	"database/sql"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"log"
	"os"
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

const (
	Healthy = iota
	Escalation
	Acknowledged
	Degraded
	UnknownStatus
)

type ServiceType string

const (
	ServiceMonitorAgent      ServiceType = "AGENT"
	ServiceMonitorWebModules ServiceType = "Web Modules"
	ServiceMonitorSNMP       ServiceType = "Network"
)

// ServiceMonitorStatus represents the current status of a monitored service
type ServiceMonitorStatus struct {
	Id                int         `json:"service_id"`
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

// GetEnvWithDefault retrieves an environment variable with a fallback default value
func GetEnvWithDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// isValidCron checks if a given string is a valid cron expression
func isValidCron(expr string) bool {
	_, err := cron.ParseStandard(expr) // Uses the standard 5-field cron format
	return err == nil
}

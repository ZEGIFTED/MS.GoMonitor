package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/CalebBoluwade/messaging"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/microsoft/go-mssqldb" // SQL Server driver
	"github.com/robfig/cron/v3"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// EnvConfig holds all environment variables
type EnvConfig struct {
	Port       string
	Host       string
	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string
	APIKey     string
	APISecret  string
	GoEnv      string
	Debug      bool
}

// LoadConfig loads environment variables from .env file
func LoadConfig() (*EnvConfig, error) {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: .env file not found. Using system environment variables")
	}

	config := &EnvConfig{
		Port:       getEnvWithDefault("PORT", "8080"),
		Host:       getEnvWithDefault("HOST", "localhost"),
		DBHost:     getEnvWithDefault("DB_HOST", "localhost"),
		DBPort:     getEnvWithDefault("DB_PORT", "1433"),
		DBName:     getEnvWithDefault("DB_NAME", "MS"),
		DBUser:     getEnvWithDefault("DB_USER", "sa"),
		DBPassword: getEnvWithDefault("DB_PASSWORD", "dbuser123$"),
		APIKey:     getEnvWithDefault("API_KEY", ""),
		APISecret:  getEnvWithDefault("API_SECRET", ""),
		GoEnv:      getEnvWithDefault("GO_ENV", "development"),
		Debug:      getEnvWithDefault("DEBUG", "false") == "true",
	}

	return config, nil
}

// getEnvWithDefault retrieves an environment variable with a fallback default value
func getEnvWithDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

type ServiceType string

const (
	ServiceMonitorAgent ServiceType = "agent"
	ServiceMonitorSNMP  ServiceType = "snmp"
)

type ServiceMonitorConfig struct {
	Id             uuid.UUID `json:"AppID"`
	Name           string
	Host           string
	Port           int
	VP             bool // Is monitoring active?
	Device         ServiceType
	RetryCount     int
	Configuration  map[string]interface{} // Settings for this service
	CheckInterval  string                 `json:"check_interval"`
	HealthCheckURL string                 `json:"health_check_url"`
	SnoozeUntil    time.Time              `json:"snooze_until"`
}

const (
	_ = iota
	Healthy
	Escalation
	Acknowledged
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
	db             *sql.DB                          // Database connection
	services       []ServiceMonitorConfig           // List of services to monitor
	statusTracking map[string]*ServiceMonitorStatus // Current status of each service
	mu             sync.RWMutex                     // For thread safety
	logger         *log.Logger                      // For logging
	checkers       map[ServiceType]ServiceChecker   // Different types of checks
	cron           *cron.Cron
	ctx            context.Context
	cancel         context.CancelFunc
}

//func checkService(service ServiceType) {}

type ServiceChecker interface {
	Check(config ServiceMonitorConfig) (bool, string)
}

type AgentServiceChecker struct{}
type SNMPServiceChecker struct{}

func (service *AgentServiceChecker) Check(config ServiceMonitorConfig) (bool, string) {
	// Get the URL from configuration
	host := config.Host

	if host == "" {
		return false, "Invalid URL configuration"
	}

	port := config.Port
	//port, ok := config.Configuration["port"].(float64)
	//if !port {
	//	return false, "Invalid port configuration"
	//}

	log.Println("Calling Agent API", host)
	agentAddress := fmt.Sprintf("%s:%d", host, int(port))

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(agentAddress)
	if err != nil {
		return false, fmt.Sprintf("HTTP check failed: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, fmt.Sprintf("Bad status code: %d", resp.StatusCode)
	}

	//service.LastCheckTime = service.LastCheckTime.Add(1 * time.Minute)
	//check.LastCheckTime = time.Now()
	//service.Check()

	return true, fmt.Sprintf("HTTP Status: %d", resp.StatusCode)
}

func (service *SNMPServiceChecker) Check(config ServiceMonitorConfig) (bool, string) {
	return true, config.HealthCheckURL
}

// NewServiceMonitor creates a new service monitor instance
func NewServiceMonitor(db *sql.DB) *ServiceMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	monitor := &ServiceMonitor{
		db:             db,
		statusTracking: make(map[string]*ServiceMonitorStatus),
		logger:         log.New(os.Stdout, "SERVICE_MONITOR: ", log.Ldate|log.Ltime|log.Lshortfile),
		checkers:       make(map[ServiceType]ServiceChecker),
		ctx:            ctx,
		cancel:         cancel,
		cron: cron.New(cron.WithChain(
			cron.SkipIfStillRunning(cron.DefaultLogger),
		)),
	}

	// Register service type checkers
	monitor.checkers[ServiceMonitorAgent] = &AgentServiceChecker{}
	monitor.checkers[ServiceMonitorSNMP] = &SNMPServiceChecker{}

	return monitor
}

func (sm *ServiceMonitor) loadServicesFromDatabase() error {
	query := `EXEC ServiceReport @SERVICE_LEVEL = 'ALL', @VP = 1;`

	//rows, err := sm.db.Query(query)
	rows, err := sm.db.QueryContext(context.Background(), query)
	if err != nil {
		return fmt.Errorf("error querying services: %v", err)
	}
	defer rows.Close()

	var services []ServiceMonitorConfig

	for rows.Next() {
		var service ServiceMonitorConfig
		var configJSON string

		err := rows.Scan(
			&service.Id,
			&service.Name,
			&service.Device,
			&configJSON,
			&service.CheckInterval,
			&service.Host,
			&service.VP,
		)

		if err != nil {
			return fmt.Errorf("error scanning service row: %v", err)
		}

		//Parse configuration JSON
		if err := json.Unmarshal([]byte(configJSON), &service.Configuration); err != nil {
			sm.logger.Printf("Warning: could not parse configuration for service %s: %v", service.Name, err)
			continue
		}

		services = append(services, service)
	}

	sm.mu.Lock()
	sm.services = services
	sm.mu.Unlock()

	sm.logger.Printf("Loaded %d active services", len(services))
	return nil
}

// checkService monitors a single service
func (sm *ServiceMonitor) checkService(service ServiceMonitorConfig) {
	sm.mu.Lock()
	status, exists := sm.statusTracking[service.Name]
	if !exists {
		status = &ServiceMonitorStatus{
			Name:          service.Name,
			Device:        service.Device,
			LiveCheckFlag: 3,
			LastCheckTime: time.Now(),
			FailureCount:  0,
		}
		sm.statusTracking[service.Name] = status
	}
	sm.mu.Unlock()

	// Get the appropriate checker for this service type
	checker, ok := sm.checkers[service.Device]
	if !ok {
		sm.logger.Printf("No checker found for service type: %s", service.Device)
		return
	}

	// Perform the service check
	isHealthy, message := checker.Check(service)

	sm.mu.Lock()
	defer sm.mu.Unlock()

	status.LastCheckTime = time.Now()

	if !isHealthy {
		status.Status = "failed"
		status.LiveCheckFlag = 1
		status.FailureCount++
		status.LastErrorLog = message

		// Log service failure
		sm.logServiceFailure(service, status)

		// Send alert if configured
		if service.VP {
			sm.sendAlert(service, status)
		}

		sm.logger.Printf("Service %s failed. Message: %s", service.Name, message)
	} else {
		status.LiveCheckFlag = 0
		status.Status = "Healthy"
		status.FailureCount = 0
		status.LastErrorLog = ""
		status.LastServiceUpTime = time.Now()

		sm.logger.Printf("Service %s is healthy. %s", service.Name, message)
	}
}

// logServiceFailure logs the service failure
func (sm *ServiceMonitor) logServiceFailure(service ServiceMonitorConfig, status *ServiceMonitorStatus) {
	// Implement database logging logic
	sm.logger.Printf("Service Failure - Name: %s, Type: %s, Error: %s",
		service.Name, service.Device, status.LastErrorLog)
}

// sendAlert sends an alert to a configured webhook
func (sm *ServiceMonitor) sendAlert(service ServiceMonitorConfig, status *ServiceMonitorStatus) {
	// Implement alert sending logic (e.g., HTTP POST to webhook)
	sm.logger.Printf("ALERT: Service %s failed. Type: %s, Failure count: %d",
		service.Name, service.Device, status.FailureCount)
}

// Start begins the monitoring process
func (sm *ServiceMonitor) Start() error {
	// Load initial services
	if err := sm.loadServicesFromDatabase(); err != nil {
		sm.logger.Fatalf("Failed to load initial services: %v", err)
	}

	// Schedule checks for each service
	// Schedule service checks
	for _, service := range sm.services {
		serviceCopy := service
		interval := serviceCopy.CheckInterval
		if interval == "" {
			interval = "*/5 * * * *"
		}

		_, err := sm.cron.AddFunc(interval, func() {
			// Check if context is cancelled before running check
			if sm.ctx.Err() != nil {
				return
			}
			sm.checkService(serviceCopy)
		})

		if err != nil {
			sm.logger.Printf("Failed to schedule service %s: %v", serviceCopy.Name, err)
		}
	}

	sm.cron.Start()
	return nil
}

func (sm *ServiceMonitor) Stop() {
	sm.cancel() // Cancel context

	// Stop the cron scheduler
	ctx := sm.cron.Stop()

	// Wait for running jobs to complete (with timeout)
	select {
	case <-ctx.Done():
		sm.logger.Println("All jobs completed")
	case <-time.After(30 * time.Second):
		sm.logger.Println("Shutdown timed out waiting for jobs")
	}
}

func main() {
	// Ensure multi-core utilization
	runtime.GOMAXPROCS(runtime.NumCPU())

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	config, err := LoadConfig()
	if err != nil {
		log.Fatal("Error loading config:", err)
	}

	// Database configuration
	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%s;database=%s;",
		config.DBHost, config.DBUser, config.DBPassword, config.DBPort, config.DBName)

	db, err := sql.Open("sqlserver", connString)
	if err != nil {
		log.Fatal("Error creating connection pool: ", err.Error())
	}
	defer db.Close()

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}

	// Create and start monitor
	monitor := NewServiceMonitor(db)
	if err := monitor.Start(); err != nil {
		log.Fatalf("Failed to start monitor: %v", err)
	}

	// Wait for shutdown signal
	<-shutdown
	log.Println("Shutting down...")

	// Graceful shutdown
	monitor.Stop()
	log.Println("Shutdown complete")

	// Implement graceful shutdown
	// Give some time for ongoing checks to complete
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("*********** SERVICE STARTED ****************", connString)

	// Keep the program running
	select {}
}

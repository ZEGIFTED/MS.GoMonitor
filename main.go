package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gosnmp/gosnmp"
	"github.com/joho/godotenv"
	_ "github.com/microsoft/go-mssqldb" // SQL Server driver
	"github.com/robfig/cron/v3"
	"io"
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
	ServiceMonitorAgent ServiceType = "AGENT"
	ServiceMonitorSNMP  ServiceType = "Network"
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
	Configuration     ServiceMonitorConfig
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

	agentAddress := fmt.Sprintf("http://%s:%d/api/v1/agent/health", host, port)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	log.Println("Calling Agent API", agentAddress)
	resp, err := client.Get(agentAddress)
	if err != nil {
		return false, fmt.Sprintf("HTTP check failed: %v", err)
	}

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)

	if err != nil {
		return false, fmt.Sprintf("Bad status code: %d, failed to read body: %v", resp.StatusCode, err)
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Sprintf("Bad status code: %d, %s", resp.StatusCode, string(bodyBytes))
	}

	//service.LastCheckTime = service.LastCheckTime.Add(1 * time.Minute)
	//check.LastCheckTime = time.Now()
	//service.Check()

	return true, fmt.Sprintf("HTTP Status: %d. Response: %s", resp.StatusCode, string(bodyBytes))
}

func (service *SNMPServiceChecker) Check(config ServiceMonitorConfig) (bool, string) {
	host := config.Host

	if host == "" {
		return false, "Invalid URL configuration"
	}

	// SNMPMetric represents an SNMP OID and its description
	type SNMPMetric struct {
		OID         string
		Description string
	}

	// Common SNMP OIDs for system information
	var commonMetrics = []SNMPMetric{
		{".1.3.6.1.2.1.1.1.0", "System Description"},
		{".1.3.6.1.2.1.1.3.0", "Uptime"},
		{".1.3.6.1.2.1.1.5.0", "System Name"},
		{".1.3.6.1.2.1.2.1.0", "Number of Interfaces"},
		{".1.3.6.1.2.1.25.2.2.0", "Memory"},
	}

	CommunityString := config.Configuration["community"].(string)

	if CommunityString == "" {
		CommunityString = "public"
	}

	// Configure SNMP connection
	snmp := &gosnmp.GoSNMP{
		Target:    host, // Replace with your device's IP
		Port:      161,
		Community: CommunityString, // Replace with your SNMP community string
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(10) * time.Second,
		Retries:   3,
	}

	// Connect to the device
	err := snmp.Connect()
	if err != nil {
		log.Fatalf("Error connecting to device: %v", err)
	}
	defer snmp.Conn.Close()

	// Get SNMP metrics
	fmt.Printf("Device Information for %s:\n", config.Host)
	fmt.Println("----------------------------------------")

	for _, metric := range commonMetrics {
		result, err := snmp.Get([]string{metric.OID})
		if err != nil {
			log.Printf("Error getting %s: %v\n", metric.Description, err)
			continue
		}

		// Display the result
		for _, variable := range result.Variables {
			fmt.Printf("OID: %s\n", variable.Name)
			switch variable.Type {
			case gosnmp.OctetString:
				fmt.Printf("Value: %s\n", string(variable.Value.([]byte)))
			case gosnmp.TimeTicks:
				fmt.Printf("TimeTicks: %d\n", gosnmp.ToBigInt(variable.Value))
			default:
				fmt.Printf("Value: %v\n", variable.Value)
			}
		}
	}

	// Get interface information
	interfaces, err := getInterfaces(snmp)
	if err != nil {
		return false, fmt.Sprintf("error getting interfaces: %v", err)
	}

	fmt.Println("\nInterface Information:")
	fmt.Println("----------------------------------------")
	for _, iface := range interfaces {
		fmt.Printf("Interface: %s\n", iface)
	}

	return true, config.HealthCheckURL
}

func getInterfaces(snmp *gosnmp.GoSNMP) ([]string, error) {
	var interfaces []string

	//err_ := snmp.Walk("1.3.6.1.2.1.2.2.1", func(variable gosnmp.SnmpPDU) error {
	//	fmt.Printf("OID: %s, Value: %v\n", variable.Name, variable.Value)
	//	return nil
	//})
	//if err_ != nil {
	//	log.Fatalf("Error walking SNMP table: %v", err_)
	//}

	// Get interface descriptions
	// Define the walk function
	walkFn := func(pdu gosnmp.SnmpPDU) error {
		if pdu.Type == gosnmp.OctetString {
			interfaces = append(interfaces, string(pdu.Value.([]byte)))
		}
		return nil
	}

	// Get interface descriptions using BulkWalk with callback
	err := snmp.BulkWalk(".1.3.6.1.2.1.2.2.1.2", walkFn)
	if err != nil {
		return nil, fmt.Errorf("BulkWalk error: %v", err)
	}

	return interfaces, nil
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
	query := `EXEC ServiceReport @SERVICE_LEVEL = 'MONITOR', @VP = 1;`

	//rows, err := sm.db.Query(query)
	rows, err := sm.db.QueryContext(context.Background(), query)
	if err != nil {
		return fmt.Errorf("error querying services: %v", err)
	}
	defer rows.Close()

	var services []ServiceMonitorConfig

	for rows.Next() {
		var service ServiceMonitorConfig
		var Configuration string

		err := rows.Scan(
			&service.Id,
			&service.Name,
			&service.Host,
			&service.Port,
			&service.VP,
			&service.Device,
			&service.RetryCount,
			&Configuration,
			&service.CheckInterval,
			&service.IsAcknowledged,
			&service.SnoozeUntil,
		)

		if err != nil {
			return fmt.Errorf("error scanning service row: %v", err)
		}

		if Configuration == "" {
			Configuration = "{}"
		}

		//Parse configuration JSON
		if err := json.Unmarshal([]byte(Configuration), &service.Configuration); err != nil {
			sm.logger.Printf("Warning: could not parse configuration for service %s: %v", service.Name, err)
			continue
		}

		fmt.Println(service.SnoozeUntil, service)
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
		status.FailureCount++

		if status.FailureCount > 1 {
			status.LiveCheckFlag = Escalation
		} else if status.FailureCount > 3 {
			status.LiveCheckFlag = Degraded
		}

		status.LastErrorLog = message

		// Log service failure
		sm.logServiceFailure(service, status)

		// Send alert if configured
		if service.VP && !service.IsAcknowledged {
			sm.sendAlert(service, status)
		}

		sm.logger.Printf("Service %s failed. Message: %s", service.Name, message)
	} else {
		status.LiveCheckFlag = Healthy
		status.Status = fmt.Sprintf("Service %s is healthy. %s", service.Name, message)
		status.FailureCount = 0
		status.LastErrorLog = ""
		status.LastServiceUpTime = time.Now()

		sm.logger.Printf("Service %s is healthy. %s", service.Name, message)
	}

	// Implement database logging logic

	_, err := sm.db.Exec(`
	UPDATE [dbo].[SystemMonitor] 
	SET 
		Status = ?, 
		LiveCheckFlag = ?, 
		LastServiceUpTime = ?, 
		LastCheckTime = ?, 
		FailureCount = ?, 
-- 		RetryCount = ? 
	WHERE 
		Name = ?`,
		status.Status,
		status.LiveCheckFlag,
		status.LastServiceUpTime,
		status.LastCheckTime,
		status.FailureCount,
		//status.RetryCount,
		service.Name,
	)

	if err != nil {
		sm.logger.Printf("Error updating SystemMonitor: %v\n", err)
	}
}

// logServiceFailure logs the service failure
func (sm *ServiceMonitor) logServiceFailure(service ServiceMonitorConfig, status *ServiceMonitorStatus) {

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
			interval = "*/1 * * * *"
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

	// Example usage
	tsData := []TimeSeriesData{
		{Timestamp: 1672531200000, Value: 95.0},
		{Timestamp: 1672531260000, Value: 98.0},
		{Timestamp: 1672531320000, Value: 97.0},
		{Timestamp: 1672531380000, Value: 96.0},
		{Timestamp: 1672531440000, Value: 99.0},
		{Timestamp: 1672531500000, Value: 100.0},
		{Timestamp: 1672531560000, Value: 101.0},
		{Timestamp: 1672531620000, Value: 102.0},
		{Timestamp: 1672531680000, Value: 103.0},
		{Timestamp: 1672531740000, Value: 104.0},
	}

	thresholds := checkTSDataAboveThreshold("CPU_Usage", "Server_1", tsData, 95.0, 10)
	fmt.Println("Thresholds breached:", thresholds)

	// Database configuration
	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%s;database=%s;",
		config.DBHost, config.DBUser, config.DBPassword, config.DBPort, config.DBName)

	fmt.Println("*********** SERVICE STARTED ****************")
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

	// Keep the program running
	select {}
}

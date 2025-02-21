package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/ZEGIFTED/MS.GoMonitor/monitors"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/utils"
	"github.com/joho/godotenv"
	_ "github.com/microsoft/go-mssqldb" // SQL Server driver
	"github.com/robfig/cron/v3"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

// EnvConfig holds all environment variables
type EnvConfig struct {
	Port      string
	Host      string
	APIKey    string
	APISecret string
	GoEnv     string
	Debug     bool
}

// LoadConfig loads environment variables from .env file
func LoadConfig() (*EnvConfig, error) {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: .env file not found. Using system environment variables")
	}

	config := &EnvConfig{
		Port:      utils.GetEnvWithDefault("PORT", "8080"),
		Host:      utils.GetEnvWithDefault("HOST", "localhost"),
		APIKey:    utils.GetEnvWithDefault("API_KEY", ""),
		APISecret: utils.GetEnvWithDefault("API_SECRET", ""),
		GoEnv:     utils.GetEnvWithDefault("GO_ENV", "development"),
		Debug:     utils.GetEnvWithDefault("DEBUG", "false") == "true",
	}

	return config, nil
}

// NewServiceMonitor creates a new service monitor instance
func NewServiceMonitor(db *sql.DB) *monitors.ServiceMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	logger := cron.PrintfLogger(utils.CronLogger)

	monitor := &monitors.ServiceMonitor{
		Db:             db,
		StatusTracking: make(map[string]*monitors.ServiceMonitorStatus),
		Logger:         utils.Logger,
		Checkers:       make(map[monitors.ServiceType]monitors.ServiceChecker),
		Ctx:            ctx,
		Cancel:         cancel,
		Cron: cron.New(
			cron.WithChain(
				cron.SkipIfStillRunning(cron.DefaultLogger),
			),
			cron.WithLogger(logger),
		),
	}

	// Register service type checkers
	monitor.Checkers[monitors.ServiceMonitorAgent] = &monitors.AgentServiceChecker{}
	monitor.Checkers[monitors.ServiceMonitorWebModules] = &monitors.WebModulesServiceChecker{}
	//monitor.Checkers[monitors.ServiceMonitorSNMP] = &monitors.SNMPServiceChecker{}

	//// Example usage
	//tsData := []TimeSeriesData{
	//	{Timestamp: 1672531200000, Value: 95.0},
	//	{Timestamp: 1672531260000, Value: 98.0},
	//	{Timestamp: 1672531320000, Value: 97.0},
	//	{Timestamp: 1672531380000, Value: 96.0},
	//	{Timestamp: 1672531440000, Value: 99.0},
	//	{Timestamp: 1672531500000, Value: 100.0},
	//	{Timestamp: 1672531560000, Value: 101.0},
	//	{Timestamp: 1672531620000, Value: 102.0},
	//	{Timestamp: 1672531680000, Value: 103.0},
	//	{Timestamp: 1672531740000, Value: 104.0},
	//}
	//
	//thresholds := CheckTSDataAboveThreshold("CPU_Usage", "Server_1", tsData, 95.0, 10)
	//fmt.Println("Thresholds breached:", thresholds)

	return monitor
}

func main() {
	// Ensure multi-core utilization
	runtime.GOMAXPROCS(runtime.NumCPU())

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	_, err := LoadConfig()
	if err != nil {
		log.Fatal("Error loading config:", err)
	}

	// Database configuration
	db := utils.DatabaseConnection()

	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			fmt.Printf("Error closing connection: %v", err)
		}
	}(db)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}

	//filePath := utils.GenerateReport(db)

	//sendTo := []string{"calebb.jnr@gmail.com", "cboluwade@nibss-plc.com.ng"}
	//messaging.SendReportEmail(sendTo, filePath)

	//extraInfo := map[string]string{
	//	"Server":       "Server-01",
	//	"Region":       "us-east-1",
	//	"CurrentValue": "95%",
	//	"Threshold":    "90%",
	//	"Timestamp":    "2025-02-11T10:30:00Z",
	//}
	//
	//slackClient := messaging.SlackBotClient()
	//slackMessage := messaging.FormatSlackMessageToSend("Test Notification", "Hello World from Go", "", "actionURL", extraInfo)
	//
	//_, err_ := slackClient.SendSlackMessage("admin_x", slackMessage)
	//if err_ != nil {
	//	return
	//}

	// Create and start monitor
	monitor := NewServiceMonitor(db)
	if err := monitor.StartService(); err != nil {
		log.Fatalf("Failed to start monitor: %v", err)
	}

	// Wait for shutdown signal
	<-shutdown
	log.Println("Shutting down...")

	// Graceful shutdown
	monitor.StopService()
	log.Println("Shutdown complete")

	// Implement graceful shutdown
	// Give some time for ongoing checks to complete
	ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Keep the program running
	//select {}
}

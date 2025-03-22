package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"

	// "log/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/ZEGIFTED/MS.GoMonitor/internal"
	"github.com/ZEGIFTED/MS.GoMonitor/monitors"
	"github.com/ZEGIFTED/MS.GoMonitor/notifier"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/messaging"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/utils"
	"github.com/joho/godotenv"
	_ "github.com/microsoft/go-mssqldb" // SQL Server driver
	"github.com/robfig/cron/v3"
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
		log.Printf("Warning: .env file not found. Using system environment variables. %s", err.Error())
	}

	config := &EnvConfig{
		Port:      constants.GetEnvWithDefault("PORT", "8082"),
		Host:      constants.GetEnvWithDefault("HOST", "localhost"),
		APIKey:    constants.GetEnvWithDefault("API_KEY", ""),
		APISecret: constants.GetEnvWithDefault("API_SECRET", ""),
		GoEnv:     constants.GetEnvWithDefault("GO_ENV", "development"),
		Debug:     constants.GetEnvWithDefault("DEBUG", "false") == "true",
	}

	return config, nil
}

func NotificationConfigurationManager(db *sql.DB) *messaging.NotificationManager {
	configManager := messaging.NotificationManager{
		Logger: utils.Logger,
		DB:     db,
		Config: &messaging.NotificationConfig{},
	}

	err := configManager.LoadConfig()
	if err != nil {
		log.Printf("Error loading config: %v", err.Error())
		return nil
	}

	configManager.Validate()

	return &configManager
}

// NewServiceMonitor creates a new service monitor instance
func NewServiceMonitor(db *sql.DB) *monitors.ServiceMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	logger := cron.PrintfLogger(utils.CronLogger)

	monitor := &monitors.ServiceMonitor{
		Db: db,
		//StatusTracking:     make(map[string]*monitors.ServiceMonitorStatus),
		//Logger:              utils.Logger,
		Checkers:            make(map[monitors.ServiceType]monitors.ServiceChecker),
		NotificationHandler: NotificationConfigurationManager(db),
		Ctx:                 ctx,
		Cancel:              cancel,
		Cron: cron.New(
			cron.WithChain(
				cron.SkipIfStillRunning(cron.DefaultLogger),
			),
			cron.WithLogger(logger),
		),

		//AlertCache: make(map[string]time.Time),
		Alerts: make(chan internal.ServiceAlertEvent, 100),
	}

	// Register service type checkers
	monitor.Checkers[monitors.ServiceMonitorAgent] = &monitors.AgentServiceChecker{}
	monitor.Checkers[monitors.ServiceMonitorWebModules] = &monitors.WebModulesServiceChecker{}
	// monitor.Checkers[monitors.ServiceMonitorSNMP] = &monitors.SNMPServiceChecker{}

	// monitor.Checkers[monitors.ServiceMonitorServer] = &monitors.ServerHealthChecker{}

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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test the connection
	err = db.PingContext(ctx)
	if err != nil {
		log.Printf("Error connecting to the database: %v", err)
	}

	go func() {
		http.HandleFunc("/ws/notifer", notifier.ServeNotifierWs)
		log.Println("Notifer server running on :2345")
		err := http.ListenAndServe(":2345", nil)
		if err != nil {
			log.Fatal("WebSocket server error:", err)
		}
	}()

	go notifier.Hub_.Run()
	// go notifier.SendNotifications()

	// Start the report generation in a goroutine
	go func() {
		ticker := time.NewTicker(7 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				nCm := NotificationConfigurationManager(db)

				if nCm != nil {
					pdfFilePath, csvFilePath := utils.GenerateReport(db)
					sendTo, err := internal.FetchReportRecipients(db)

					if err != nil {
						log.Println("Error fetching report recipients:", err)
					}

					fmt.Println("Report Receipents >>>", sendTo)

					sendTo_ := []string{"calebb.jnr@gmail.com", "cboluwade@nibss-plc.com.ng"}

					if pdfFilePath != "" && csvFilePath != "" {
						nCm.SendReportEmail(sendTo_, pdfFilePath, csvFilePath)
					} else {
						log.Println("Skipping Reports")
					}
				} else {
					log.Println("No notification configuration manager found", nCm)
				}
			case <-shutdown:
				return
			}
		}
	}()

	// Create the payload
	//teamsMessage := messaging.TeamsMessage{
	//	Type:       "MessageCard",
	//	Context:    "http://schema.org/extensions",
	//	Title:      "Test Teams Message 🚨",
	//	Text:       "Test Teams Message__ da",
	//	ThemeColor: "0076D7", // Microsoft blue color
	//}
	//
	//webhookURL := "https://outlook.office.com/webhook/YOUR_WEBHOOK_URL"
	//
	//err_ := teams.SendTeamsMessage(webhookURL, teamsMessage)
	//if err_ != nil {
	//	fmt.Println("Error:", err)
	//}
	//extraInfo := map[string]string{
	//	"Server":       "Server-01",
	//	"Region":       "us-east-1",
	//	"CurrentValue": "95%",
	//	"Threshold":    "90%",
	//	"Timestamp":    "2025-02-11T10:30:00Z",
	//}
	//

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
	_, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Keep the program running
	//select {}
}

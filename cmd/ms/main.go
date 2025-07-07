package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"

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

	"github.com/ZEGIFTED/MS.GoMonitor/pkg/plugins/http_monitor"
	sslcheck "github.com/ZEGIFTED/MS.GoMonitor/pkg/plugins/ssl_checker"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/utils"
	mstypes "github.com/ZEGIFTED/MS.GoMonitor/types"
	"github.com/joho/godotenv"
	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/robfig/cron/v3"
)

// EnvConfig holds all environment variables
// type EnvConfig struct {
// 	Port      string
// 	Host      string
// 	APIKey    string
// 	APISecret string
// 	GoEnv     string
// 	Debug     bool
// }

// LoadConfig loads environment variables from .env file
// func LoadConfig() (*EnvConfig, error) {
// 	// Load .env file
// 	err := godotenv.Load()
// 	if err != nil {
// 		log.Printf("Warning: .env file not found. Using system environment variables. %s", err.Error())
// 	}

// 	config := &EnvConfig{
// 		Port:      constants.GetEnvWithDefault("PORT", "8082"),
// 		Host:      constants.GetEnvWithDefault("HOST", "localhost"),
// 		APIKey:    constants.GetEnvWithDefault("SLACK_TOKEN", ""),
// 		APISecret: constants.GetEnvWithDefault("LOG_LEVEL", ""),
// 		GoEnv:     constants.GetEnvWithDefault("GO_ENV", "development"),
// 		Debug:     constants.GetEnvWithDefault("DEBUG", "false") == "true",
// 	}

// 	return config, nil
// }

func NotificationConfigurationManager(db *sql.DB) *messaging.NotificationManager {
	configManager := messaging.NotificationManager{
		Logger: utils.Logger,
		DB:     db,
		Config: &mstypes.NotificationConfig{},
	}

	err := configManager.LoadConfig()
	if err != nil {
		log.Printf("Error loading config: %v", err.Error())
	}

	if err := configManager.Validate(); err != nil {
		log.Fatalf("❌ Notification config validation failed: %v", err)
	}

	return &configManager
}

// NewServiceMonitor creates a new service monitor instance
func NewServiceMonitor(db *sql.DB) *monitors.MonitoringEngine {
	ctx, cancel := context.WithCancel(context.Background())

	logger := cron.PrintfLogger(utils.CronLogger)

	monitor := &monitors.MonitoringEngine{
		Services: make([]monitors.ServiceMonitorData, 0),
		Db:       db,
		//Logger:              utils.Logger,
		//StatusTracking:     make(map[string]*monitors.ServiceMonitorStatus),
		DefaultHealth:     &monitors.HealthCheck{},
		Plugins:           make(map[string]monitors.ServiceMonitorPlugin),
		PluginInitialized: make(map[string]bool),

		NotificationHandler: NotificationConfigurationManager(db),
		Ctx:                 ctx,
		Cancel:              cancel,
		Cron: cron.New(
			cron.WithChain(
				cron.SkipIfStillRunning(cron.DefaultLogger),
			),
			cron.WithLogger(logger),
		),
		Alerts: make(chan internal.ServiceAlertEvent, 100),
	}

	// Register Default Service checkers
	// monitor.Checkers[monitors.ServiceMonitorWebModules] = &monitors.WebModulesServiceChecker{}
	// // monitor.Checkers[monitors.ServiceMonitorSNMP_V3] = &monitors.SNMPServiceCheckerV3{}
	// monitor.Checkers[monitors.ServiceMonitorSNMP] = &monitors.SNMPServiceChecker{}

	// monitor.Checkers[monitors.ServiceMonitorServer] = &monitors.ServerHealthChecker{}

	monitor.Plugins["http_monitor"] = &http_monitor.HTTPMonitorPlugin{}
	monitor.Plugins["ssl_check"] = &sslcheck.SSLChecker{}

	if false {
		// monitor.Checkers[monitors.ServiceMonitorAgent] = &monitors.AgentServiceChecker{}
	}

	return monitor
}

func main() {
	// 	err := godotenv.Load()
	// 	if err != nil {
	// 		log.Printf("Warning: .env file not found. Using system environment variables. %s", err.Error())
	// 	}

	envErr := godotenv.Load()

	if envErr != nil {
		log.Fatalf("Fatal Error loading Env file. %s", envErr.Error())
	}

	// constants
	log.Println("Environment Variables Loaded")

	// Ensure multi-core utilization
	runtime.GOMAXPROCS(runtime.NumCPU())

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// _, err := LoadConfig()
	// if err != nil {
	// 	log.Fatal("Error loading config:", err)
	// }

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
	err := db.PingContext(ctx)
	if err != nil {
		log.Printf("Error connecting to the database: %v", err)
	}

	go func() {
		http.HandleFunc("/ws/notifer", notifier.ServeNotifierWs)

		http.HandleFunc("/ws/management", notifier.ServeManagementInterface)
		http.HandleFunc("/ws/synthetic", notifier.ServeSyntheticDashboard)

		wsPortStr := constants.GetEnvWithDefault("WS_PORT", "2345")
		wsPortInt, err_ := strconv.Atoi(wsPortStr)
		if err_ != nil {
			wsPortInt = 1
			log.Fatalf("invalid report interval: %v", err)
		}

		log.Printf("Dashboard && Notifer server running on :%v", wsPortInt)

		err := http.ListenAndServe(fmt.Sprintf(":%d", wsPortInt), nil)
		if err != nil {
			log.Fatal("WebSocket server error:", err)
		}
	}()

	go notifier.Hub_.Run()
	go notifier.DashHub.Run()
	go notifier.BroadcastDashboardData(db)

	// Start the report generation in a goroutine
	go func() {
		reportStr := constants.GetEnvWithDefault("REPORT_HOUR_INTERVAL", "1")
		reportInt, err := strconv.Atoi(reportStr)
		if err != nil {
			reportInt = 1
			log.Fatalf("invalid report interval: %v", err)
		}
		ticker := time.NewTicker(time.Duration(reportInt) * time.Hour)
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

	// handler := &monitors.NetworkManager{
	// 	Community: "public",
	// 	TrapPort:  162, // Standard SNMP trap port
	// }

	// trapErr := handler.StartListener()
	// if err != nil {
	// 	log.Fatalf("Failed to start trap handler: %v", trapErr)
	// }

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

	if err := monitor.StartEngine(); err != nil {
		log.Fatalf("Failed to Start Monitoring Engine: %v", err)
	}

	// Wait for shutdown signal
	<-shutdown
	log.Println("Shutting down...")
	// handler.StopListener()

	// Graceful shutdown
	monitor.StopEngine()
	log.Println("Monitoring Engine Shutdown complete")

	// Implement graceful shutdown
	// Give some time for ongoing checks to complete
	_, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Keep the program running
	//select {}
}

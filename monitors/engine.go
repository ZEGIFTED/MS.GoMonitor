package monitors

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"plugin"
	"strconv"
	"sync"
	"time"

	"github.com/ZEGIFTED/MS.GoMonitor/internal"
	"github.com/ZEGIFTED/MS.GoMonitor/notifier"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/utils"
	mstypes "github.com/ZEGIFTED/MS.GoMonitor/types"
	"github.com/google/uuid"

	"github.com/lib/pq" // PostgreSQL driver
)

// StartService begins the monitoring process
func (sm *MonitoringEngine) StartEngine() error {
	log.Println("Starting MS Monitoring Engine...")

	// Load all plugins
	if err := sm.LoadPlugins(); err != nil {
		return fmt.Errorf("failed to load plugins: %v", err)
	}

	// Load initial services
	if err := sm.LoadServiceInventory(); err != nil {
		log.Fatalf("Failed to load initial services: %v", err)
	}

	// Schedule service checks for each service
	for _, service := range sm.Services {
		serviceCopy := service
		interval := serviceCopy.CheckInterval

		if interval == "" || !utils.IsValidCron(interval) {
			interval = constants.DefaultCronExpression
		}

		_, err := sm.Cron.AddFunc(interval, func() {
			// Check if context is cancelled before running check
			if sm.Ctx.Err() != nil {
				return
			}
			sm.CheckServices(serviceCopy)
		})

		if err != nil {
			log.Printf("Failed to schedule service %s: %v", serviceCopy.Name, err)
		}
	}

	sm.Cron.Start()
	sm.AlertHandler()
	return nil
}

// StopService begins the monitoring process
func (sm *MonitoringEngine) StopEngine() {
	sm.Cancel() // Cancel context

	log.Printf("Stopping MS-SVC_MONITOR")

	// Stop the cron scheduler
	ctx := sm.Cron.Stop()
	close(sm.Alerts)

	// Cleanup all plugins
	sm.MU.RLock()
	plugins := make([]ServiceMonitorPlugin, 0)
	for _, plugin := range sm.Plugins {
		plugins = append(plugins, plugin)
	}
	sm.MU.RUnlock()

	for _, plugin := range plugins {
		if err := plugin.Cleanup(); err != nil {
			log.Printf("Error cleaning up plugin %s: %v", plugin.Name(), err)
		}
	}

	// Wait for running jobs to complete (with timeout)
	select {
	case <-ctx.Done():
		log.Println("All jobs completed")
	case <-time.After(15 * time.Second):
		log.Println("Shutdown timed out waiting for jobs")
	}
}

func (sm *MonitoringEngine) LoadServiceInventory() error {
	log.Println("Fetching Services...")

	// Try to load services from the database first
	services, err := sm.LoadDatabaseMonitoredServices()
	if err != nil {
		log.Printf("Failed to load services from database: %v. Falling back to JSON file...", err)

		// If database fails, load from JSON file
		services, err = sm.loadServicesFromJSON("services.json")
		if err != nil {
			return fmt.Errorf("failed to load services from JSON: %v", err)
		}
	}

	sm.MU.Lock()
	sm.Services = services
	sm.MU.Unlock()

	// Initialize plugins for all services
	for _, service := range services {
		if err := sm.initializeServicePlugins(service); err != nil {
			slog.Error("Failed to initialize plugins for service %s: %v", service.Name, err)
		}
	}

	return nil
}

func (sm *MonitoringEngine) LoadDatabaseMonitoredServices() ([]ServiceMonitorData, error) {
	log.Println("Fetching Services From Database...")

	query := `SELECT "SystemMonitorId",
       "ServiceName",
       "IPAddress",
       "Port",
       "IsMonitored",
	   "CurrentHealthCheck",
       "Device",
       "FailureCount",
       "RetryCount",
	   "Configuration",
       "CheckInterval",
       "IsAcknowledged",
       "SnoozeUntil",
       "Plugins",
		"AgentAPI"
		FROM servicereport('ALL', NULL, TRUE, NULL);`

	rows, err := sm.Db.QueryContext(sm.Ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error querying services: %v", err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {

		}
	}(rows)

	var services []ServiceMonitorData

	for rows.Next() {
		var service ServiceMonitorData
		var uuidStr string
		var Configuration string
		var plugins pq.StringArray // PostgreSQL returns JSON as []byte

		err := rows.Scan(
			&uuidStr,
			&service.Name,
			&service.Host,
			&service.Port,
			&service.IsMonitored,
			&service.CurrentHealthCheck,
			&service.Device,
			&service.FailureCount,
			&service.RetryCount,
			&Configuration,
			&service.CheckInterval,
			&service.IsAcknowledged,
			&service.SnoozeUntil,
			&plugins,
			&service.AgentAPIBaseURL,
		)

		if err != nil {
			return nil, fmt.Errorf("error scanning service row: %v", err)
		}

		service.SystemMonitorId, err = uuid.Parse(uuidStr)
		if err != nil {
			slog.Error(err.Error())
		}

		service.Plugins = plugins
		// if err := json.Unmarshal(pluginsRaw, &service.Plugins); err != nil {
		// 	log.Println("Error decoding plugins JSON:", err)
		// }

		// Parse plugins JSON array
		// if len(pluginJsonRaw) > 0 {
		// 	err := json.Unmarshal([]byte(pluginJsonRaw), &service.Plugins)
		// 	if err != nil {
		// 		log.Printf("Error unmarshaling plugins: %v", err)
		// 	}
		// }

		if Configuration == "" {
			Configuration = "{}"
		}

		//Parse configuration JSON
		if err := json.Unmarshal([]byte(Configuration), &service.Configuration); err != nil {
			log.Printf("Warning: could not parse configuration for service %s: %v", service.Name, err)
			continue
		}

		services = append(services, service)
	}

	sm.MU.Lock()
	sm.Services = services
	sm.MU.Unlock()

	log.Printf("Loaded %d active Services", len(services))
	return services, nil
}

func (sm *MonitoringEngine) loadServicesFromJSON(filename string) ([]ServiceMonitorData, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading JSON file: %v", err)
	}

	var services []ServiceMonitorData
	err = json.Unmarshal(file, &services)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return services, nil
}

// LoadPlugins loads all compatible plugins from the plugin directory
func (sm *MonitoringEngine) LoadPlugins() error {
	var pluginDir string
	flag.StringVar(&pluginDir, "plugin-dir", "./plugins", "Directory to load plugins from")
	flag.Parse()

	// Try multiple plugin directories
	var loadedPlugins int
	var loadErrors []string

	constants.PluginDirs = append(constants.PluginDirs, pluginDir)

	for _, dir := range constants.PluginDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			log.Printf("Plugin directory %s doesn't exist, creating it", dir)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create plugin directory: %v", err)
			}
			// return nil // No plugins to load since we just created the directory
			continue
		}

		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() || filepath.Ext(path) != ".so" {
				return nil
			}

			plug, err := plugin.Open(path)
			if err != nil {
				loadErrors = append(loadErrors, fmt.Sprintf("failed to load %s: %v", path, err))
				return nil
			}

			sym, err := plug.Lookup("Plugin")
			if err != nil {
				loadErrors = append(loadErrors,
					fmt.Sprintf("%s doesn't export 'Plugin' symbol: %v", path, err))
				return nil
			}

			monitor, ok := sym.(ServiceMonitorPlugin)
			if !ok {
				loadErrors = append(loadErrors,
					fmt.Sprintf("%s doesn't implement Service Monitor Plugin interface", path))
				return nil
			}

			if _, exists := sm.Plugins[monitor.Name()]; !exists {
				sm.Plugins[monitor.Name()] = monitor
				loadedPlugins++
				log.Printf("Loaded plugin: %s (%s) from %s",
					monitor.Name(), monitor.Description(), path)
			}

			return nil
		})

		if err != nil {
			loadErrors = append(loadErrors, fmt.Sprintf("error walking %s: %v", dir, err))
		}
	}

	if loadedPlugins == 0 && len(loadErrors) > 0 {
		return fmt.Errorf("no plugins loaded, errors: %v", loadErrors)
	}

	if len(loadErrors) > 0 {
		log.Printf("Plugin loading completed with some errors: %v", loadErrors)
	}

	return nil
}

// initializeServicePlugins sets up plugins for a specific service
func (sm *MonitoringEngine) initializeServicePlugins(service ServiceMonitorData) error {
	if len(service.Plugins) > constants.MaxPluginsPerService {
		return fmt.Errorf("service %s has too many plugins (%d), max is %d",
			service.Name, len(service.Plugins), constants.MaxPluginsPerService)
	}

	var missingPlugins []string
	var initErrors []string

	for _, pluginName := range service.Plugins {
		plugin, exists := sm.Plugins[pluginName]
		if !exists {
			missingPlugins = append(missingPlugins, pluginName)
			continue
		}

		// Verify plugin supports this service type
		supported := false
		for _, t := range plugin.SupportedTypes() {
			if t == service.Device {
				supported = true
				break
			}
		}

		if !supported {
			initErrors = append(initErrors,
				fmt.Sprintf("plugin %s doesn't support service type %s",
					pluginName, service.Device))
			continue
		}

		// Initialize with service-specific configuration
		if err := plugin.Initialize(service.Configuration); err != nil {
			initErrors = append(initErrors,
				fmt.Sprintf("failed to initialize plugin %s: %v", pluginName, err))
		}
	}

	if len(missingPlugins) > 0 {
		return fmt.Errorf("missing plugins: %v", missingPlugins)
	}

	if len(initErrors) > 0 {
		return fmt.Errorf("initialization errors: %v", initErrors)
	}

	return nil
}

func (sm *MonitoringEngine) CheckServices(service ServiceMonitorData) {
	if service.SnoozeUntil.Valid && time.Now().Before(service.SnoozeUntil.Time) {
		log.Printf("[SNOOZED] %s is snoozed until %v", service.Name, service.SnoozeUntil.Time)
		return
	}

	tx, err := sm.Db.BeginTx(sm.Ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		log.Printf("[ERROR] Starting transaction: %v", err)
		return
	}
	defer tx.Rollback()

	mainStatus, err := sm.DefaultHealth.Check(sm.Ctx, sm.Db, service)
	if err != nil {
		log.Printf("[ERROR] Default check failed for %s: %v", service.Name, err)
		sm.handleServiceFailure(service, mainStatus, err)
		return
	}

	var pluginStatuses []MonitoringResult
	for _, pluginName := range service.Plugins {
		slog.InfoContext(sm.Ctx, "Service Plugins", "PluginName", pluginName)
		plugin, ok := sm.Plugins[pluginName]
		if !ok {
			sm.handleUnknownServiceType(pluginName, service)
			continue
		}

		for _, allowedPluginType := range plugin.SupportedTypes() {
			if service.Device != allowedPluginType {
				slog.Info("Checking if Plugin Should Be Used", "Plugin", allowedPluginType, "Device", service.Device)
				continue
			}
		}

		pluginStatus, err := plugin.Check(sm.Ctx, sm.Db, service)
		if err != nil {
			// PLUGIN [%s] CHECK Failed for %s.
			sm.handleServiceFailure(service, pluginStatus, err)
		}

		pluginStatuses = append(pluginStatuses, pluginStatus)
		_ = plugin.Cleanup()
	}

	finalStatus := sm.mergeStatuses(&mainStatus, pluginStatuses)
	sm.updateDatabase(tx, uuid.New(), service, finalStatus, pluginStatuses)
	sm.StatusTracking.Store(service.Name, finalStatus)
}

func (sm *MonitoringEngine) isFailureStatus(status constants.StatusInfo) bool {
	slog.Info("Checking if Engine Should Fail Service", "Flag", status.Flag)
	return status.Flag != 1
}

func (sm *MonitoringEngine) mergeStatuses(main *MonitoringResult, plugins []MonitoringResult) MonitoringResult {
	for _, p := range plugins {
		if sm.isFailureStatus(p.HealthReport) {
			main.HealthReport = constants.GetStatusInfo(p.HealthReport.Flag, "Plugin Failure Detected")
			break
		}
	}
	return *main
}

// CheckService monitors a single service
// func (sm *MonitoringEngine) CheckService(service ServiceMonitorData) {
// 	// Skip if service is snoozed
// 	if service.SnoozeUntil.Valid && time.Now().Before(service.SnoozeUntil.Time) {
// 		log.Printf("%s is Snoozed", service.Name)
// 		return
// 	}

// 	// Start transaction
// 	tx, err := sm.Db.BeginTx(sm.Ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
// 	if err != nil {
// 		log.Printf("Error starting transaction: %v", err)
// 		return
// 	}
// 	defer tx.Rollback() // ensure rollback if not committed

// 	// Run default checker first
// 	mainStatus, err := sm.DefaultHealth.Check(sm.Ctx, sm.Db, service)
// 	if err != nil {
// 		log.Printf("Default Health Check failed for %s: %v ------- %v", service.Name, err, mainStatus)
// 		sm.handleServiceFailure(service, mainStatus, err)
// 		return
// 	}

// 	// 2. Prepare to collect plugin statuses
// 	var pluginStatuses []MonitoringResult

// 	// Run all configured plugins
// 	for _, pluginName := range service.Plugins {
// 		plugin, exists := sm.Plugins[pluginName]
// 		if !exists {
// 			sm.handleUnknownServiceType(pluginName, service)
// 			continue
// 		}

// 		pluginStatus, err := plugin.Check(sm.Ctx, sm.Db, service)
// 		slog.Info("PLugin S+++++++++++++", "ds", pluginStatus)
// 		if err != nil {
// 			sm.handleServiceFailure(service, pluginStatus, err)
// 			// continue
// 		} else {
// 			sm.handleServiceRecovery(service, &pluginStatus)
// 		}
// 		// mainStatus = sm.mergeStatuses(mainStatus, pluginStatus)

// 		pluginStatuses = append(pluginStatuses, pluginStatus)
// 	}

// 	// Database logging logic to Update the database
// 	sm.updateDatabase(tx, uuid.New(), service, mainStatus, pluginStatuses)

// 	// Update the service status in sync.Map
// 	sm.StatusTracking.Store(service.Name, mainStatus)

// 	// Call the MetricEngine
// 	//monitors.MetricEngine()
// }

func (sm *MonitoringEngine) handleUnknownServiceType(pluginName string, service ServiceMonitorData) {
	sm.MU.Lock()
	defer sm.MU.Unlock()

	log.Printf("PLUGIN [%s] not found for service -> %s", pluginName, service.Name)
	service.FailureCount++

	currentStatus := &MonitoringResult{
		Name:          service.Name,
		HealthReport:  constants.GetStatusInfo(constants.InvalidConfiguration, "Failed Monitor Check; No Monitor Impl found for Service type"),
		LastCheckTime: time.Now(),
	}

	sm.StatusTracking.Store(service.Name, currentStatus)
}

func (sm *MonitoringEngine) handleServiceFailure(service ServiceMonitorData, currentStatus MonitoringResult, err error) {
	// Log service failure
	serviceAlertIdentifier := service.SystemMonitorId.String() + "|" + service.Name
	slog.Error("Service Check Failed",
		"service", service.Name,
		"error", err,
		"alert_identifier", serviceAlertIdentifier,
	)

	// Update status
	currentStatus.FailureCount++

	// Determine escalation level
	if currentStatus.FailureCount > 3 {
		currentStatus.HealthReport = constants.GetStatusInfo(constants.Degraded, err.Error())
	} else if currentStatus.FailureCount > 1 {
		currentStatus.HealthReport = constants.GetStatusInfo(constants.Escalation, err.Error())
	}

	// Throttle alerts to avoid alert fatigue
	if lastAlert, ok := sm.AlertCache.Load(serviceAlertIdentifier); !ok || func() bool {
		lastAlertTime, valid := lastAlert.(time.Time)
		return valid && time.Since(lastAlertTime) < constants.AlertThrottleTime // Prevent alert spam within 10-min window
	}() {
		log.Println(serviceAlertIdentifier, currentStatus.FailureCount, constants.FailureThresholdCount, service.IsAcknowledged)
		if currentStatus.FailureCount > constants.FailureThresholdCount && !service.IsAcknowledged {
			sm.Alerts <- internal.ServiceAlertEvent{
				SystemMonitorId: service.SystemMonitorId,
				ServiceName:     service.Name,
				Message:         fmt.Sprintf("%s is down: %s", service.Name, err.Error()),
				Severity:        "critical",
				Timestamp:       time.Now(),
				AgentRepository: service.AgentRepository,
				AgentAPI:        service.AgentAPIBaseURL,
			}

			serviceAlertMessage := notifier.NotiferEvent{
				Title:      fmt.Sprintf("%s is down", service.Name),
				Identifier: serviceAlertIdentifier,
				Message:    err.Error(),
				Timestamp:  time.Now().Format(time.RFC3339),
			}

			notifier.SendNotification(serviceAlertMessage)

			log.Printf("Added Failed Service to Alert Channel: %s", serviceAlertIdentifier)
			return
		}

		sm.AlertCache.Store(serviceAlertIdentifier, time.Now())
	}
}

func (sm *MonitoringEngine) handleServiceRecovery(service ServiceMonitorData, currentStatus *MonitoringResult) *MonitoringResult {
	// Update status for healthy service
	currentStatus.HealthReport = constants.GetStatusInfo(constants.Healthy, "Service is healthy and working optimal")
	currentStatus.FailureCount = 0
	currentStatus.LastServiceUpTime = time.Now()

	// Log service recovery
	log.Printf("Service Check -> %s is healthy.", service.Name)

	return currentStatus
}

// func (sm *MonitoringEngine) updateDatabase(service ServiceMonitorData, currentStatus ServiceMonitorStatus) {
// 	// Update the database with the latest status
// 	_, err := sm.Db.Exec(`
//         UPDATE [dbo].[SystemMonitor]
//         SET
//             Status = @Status,
//             LiveCheckFlag = @LiveCheckFlag,
//             LastServiceUpTime = @LastServiceUpTime,
//             LastCheckTime = @LastCheckTime,
//             FailureCount = @FailureCount
//         WHERE
//             ServiceName = @Name`,
// 		sql.Named("Status", currentStatus.Status),
// 		sql.Named("LiveCheckFlag", currentStatus.LiveCheckFlag),
// 		sql.Named("LastServiceUpTime", currentStatus.LastServiceUpTime),
// 		sql.Named("LastCheckTime", currentStatus.LastCheckTime),
// 		sql.Named("FailureCount", currentStatus.FailureCount),
// 		sql.Named("Name", service.Name),
// 	)

// 	if err != nil {
// 		log.Printf("Error updating SystemMonitor for service %s: %v", service.Name, err)
// 	}
// }

func (sm *MonitoringEngine) updateDatabase(tx *sql.Tx, resultID uuid.UUID, service ServiceMonitorData, currentStatus MonitoringResult, pluginStatuses []MonitoringResult) error {
	// Update the database with the latest status using PostgreSQL and quoted identifiers
	_, err := tx.ExecContext(sm.Ctx, `INSERT INTO "MonitoringResultHistory" ("Id", "SystemMonitorId", "HealthReport", "ExecutionTime", "Status") VALUES ($1, $2, $3, $4, $5)
	`, resultID, service.SystemMonitorId, currentStatus.HealthReport.Description, currentStatus.LastCheckTime, strconv.Itoa(currentStatus.HealthReport.Flag))

	if err != nil {
		log.Printf("Error inserting monitoring result: %v", err)
		return err
	}
	_, errc := sm.Db.Exec(`
        UPDATE "SystemMonitor"
        SET 
			"CurrentHealthCheck" = $5,
            "LastServiceUpTime" = $1, 
            "LastCheckTime" = $2, 
            "FailureCount" = $3
        WHERE 
            "ServiceName" = $4`,
		currentStatus.LastServiceUpTime,
		currentStatus.LastCheckTime,
		currentStatus.FailureCount,
		service.Name,
		service.CurrentHealthCheck,
	)

	if errc != nil {
		log.Printf("Error updating SystemMonitor for service %s: %v", service.Name, err)
	}

	if len(pluginStatuses) > 0 {
		stmt, err := tx.PrepareContext(sm.Ctx, `
		INSERT INTO "PluginMonitoringResults" ("Id", "MonitoringResultId", "ServicePluginId", "Status", "HealthReport"
		) VALUES ($1, $2, $3, $4, $5)
	`)
		if err != nil {
			log.Printf("Prepare statement failed: %v", err)
			return err
		}
		defer stmt.Close()

		for _, ps := range pluginStatuses {
			_, err := stmt.ExecContext(sm.Ctx, uuid.New(), resultID, ps.ServicePluginID, ps.HealthReport.Name, ps.HealthReport.Description)
			if err != nil {
				log.Printf("Failed to insert plugin result (%s): %v", ps.Name, err)
				return err
			}
		}

	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		slog.ErrorContext(sm.Ctx, "failed to commit transaction: %w", "Error", err)
	}

	return nil
}

//
//// TrackServiceFailure logs the service failure
//func (sm *MonitoringEngine) TrackServiceFailure(service ServiceMonitorData, status *ServiceMonitorStatus, severity string) {
//
//	if !service.IsAcknowledged && status.FailureCount > 2 && utils.IsValidUUID(service.SystemMonitorId.String()) {
//		log.Printf("Escalating Service -> %s", service.Name)
//
//		//sm.Db.Exec(`INSERT INTO [dbo].[AgentEscalations] (AgentID, Metric, Status, RootCause, Escalation, ENTITY_HOST)`, ("", status.Status, status.LastErrorLog, service.Host, service.Id, status.FailureCount))
//
//	} else if !utils.IsValidUUID(service.SystemMonitorId.String()) {
//		log.Printf("Service Not linked to an app -> %s", service.SystemMonitorId, service.Name)
//		//
//		//d := sm.NotificationHandler
//		//d.LoadConfig()
//		//
//		//d.GetEmailConfig()
//
//	} else {
//		log.Printf("Service Failure - %s -> Name: %s, Type: %s, Error: %s, IsAcknowledged: %t, FailureCount: %d, App: %t",
//			service.SystemMonitorId, service.Name, service.Device, status.LastErrorLog, service.IsAcknowledged, status.FailureCount, utils.IsValidUUID(service.SystemMonitorId.String()))
//
//	}
//}

func (sm *MonitoringEngine) GetUnprocessedAlertsCount() int {
	sm.MU.RLock()         // Lock for read-only access
	defer sm.MU.RUnlock() // Unlock after function execution
	return len(sm.Alerts)
}

// AlertHandler checks if an alert should be sent (rate-limiting) and sends them.
func (sm *MonitoringEngine) AlertHandler() {
	go func() {
		for {
			select {

			case alert, ok := <-sm.Alerts:
				if !ok {
					log.Println("Alert Channel Closed, Stopping Alert Handler.")
					return
				}

				log.Printf("Processing alert for service: %s", alert.ServiceName)

				// Collect all SystemMonitorIds from the alert channel
				var systemMonitorIds []string
				var serviceNames []string
				systemMonitorIds = append(systemMonitorIds, alert.SystemMonitorId.String())
				serviceNames = append(serviceNames, alert.ServiceName)

				// Fetch recipients for this alert
				recipientMap, err := internal.FetchUsersAndGroupsByServiceNames(sm.Ctx, sm.Db, systemMonitorIds, serviceNames)
				if err != nil {
					log.Printf("Error fetching recipients for service %s: %v", alert.ServiceName, err)
					continue
				}

				alertIdentifier := alert.SystemMonitorId.String() + "|" + alert.ServiceName

				recipients, exists := recipientMap[alertIdentifier]
				if !exists {
					//if !exists || len(recipients.Users) == 0 {
					log.Printf("No recipients found for service Identifier %s", alertIdentifier)
					continue
				}

				currentStatus, ok := sm.StatusTracking.Load(alert.ServiceName)

				if !ok {
					log.Println(err)
				}

				if currentStatus != nil {
					fmt.Println("currentStatus in Alert", currentStatus)
				}

				if alert.Device != "Network" && alert.Device != "Database" {
					agentHttpClient, agentStatsEndpoint, aErr := alert.AgentRepository.ValidateAgentURL(alert.AgentAPI, "/api/v1/agent/resource-usage?limit=5")

					if aErr != nil {
						log.Println(aErr.Error())
					}

					stats, statsErr := alert.AgentRepository.GetAgentServiceStats(agentHttpClient, agentStatsEndpoint)

					if statsErr != nil {
						log.Println(statsErr.Error())
					}

					log.Println(stats)
					alert.ServiceStats = stats
				}

				// Send the notification asynchronously
				go func(alert internal.ServiceAlertEvent, recipients mstypes.NotificationRecipients) {
					err := sm.SendDowntimeServiceNotification(alert, recipients)
					if err != nil {
						log.Printf("Failed to send alert for service %s: %v", alert.ServiceName, err)
					} else {
						log.Printf("Successfully sent alert for service %s", alert.ServiceName)
					}
				}(alert, recipients)
			}
		}
	}()
}

func (sm *MonitoringEngine) SendDowntimeServiceNotification(event internal.ServiceAlertEvent, recipients mstypes.NotificationRecipients) error {
	log.Printf("Processing alert for service: %s", event.ServiceName)

	emailConfig := sm.NotificationHandler.GetEmailConfig()
	slackConfig := sm.NotificationHandler.GetSlackConfig()

	var recipientsGroup = internal.GroupRecipientsByPlatform(recipients.Users)

	var wg sync.WaitGroup
	var formattedEmails []string
	errChan := make(chan error, len(recipients.Users))

	// Send notifications to users
	for platform, recipientsList := range recipientsGroup {
		wg.Add(1)

		go func(r []mstypes.NotificationRecipient) {
			defer wg.Done()

			switch platform {
			case "Email":
				if emailConfig.Enabled {
					for _, user := range r {
						formattedEmail, err := sm.NotificationHandler.FormatEmailMessageToSend(event, user.UserName, user.GroupName, constants.ConsoleBaseURL, make(map[string]any))
						if err != nil {
							errChan <- err
						}

						formattedEmails = append(formattedEmails, formattedEmail)
					}

					emailAddresses := make([]string, len(r))
					for i, recipient := range r {
						emailAddresses[i] = recipient.Email
					}

					if err := sm.NotificationHandler.SendEmail(emailAddresses, event.ServiceName+" Health Check", formattedEmails); err != nil {
						log.Printf("failed to send email to %s: %v", emailAddresses, err)
						errChan <- err
					}

				}
			case "Slack":
				if slackConfig.Enabled {
					var slackClient = sm.NotificationHandler.SlackBotClient(slackConfig)
					for _, user := range r {
						slackMessage := sm.NotificationHandler.FormatSlackMessageToSend(event, user.GroupName, "", constants.ConsoleBaseURL, make(map[string]string))

						//"admin_x"
						_, err := slackClient.SendSlackMessage(user.SlackId, slackMessage)
						if err != nil {
							log.Printf("Failed to send Slack message to %s: %v", user.Email, err)
							errChan <- err
						}
					}
				}

			// Add other platforms as needed
			default:
				for _, user := range recipientsList {
					log.Printf("Unsupported platform: %s for user %s", user.Platform, user.Email)
					errChan <- fmt.Errorf("unsupported platform: %s", user.Platform)
				}
			}

		}(recipientsList)

	}

	// Wait for all notifications to complete
	wg.Wait()
	close(errChan)

	// Check if there were errors
	if len(errChan) > 0 {
		return fmt.Errorf("one or more notifications failed")
	}
	return nil
}

type TimeSeriesData struct {
	Timestamp int64
	Value     float64
}

func CheckTSDataAboveThreshold(tsData []TimeSeriesData, threshold float64, arrSequenceLength int) []struct {
	Timestamp string
	Values    []float64
} {
	var timestampArr []int64
	var valueArr []float64
	var thresholds []struct {
		Timestamp string
		Values    []float64
	}
	var sequenceData [][]float64

	// Populate timestampArr and valueArr
	for _, val := range tsData {
		timestampArr = append(timestampArr, val.Timestamp)
		valueArr = append(valueArr, val.Value)
	}

	// Group values into sequences
	for i := 0; i < len(valueArr); i += arrSequenceLength {
		if i+arrSequenceLength <= len(valueArr) {
			sequence := valueArr[i : i+arrSequenceLength]
			sequenceData = append(sequenceData, sequence)
		}
	}

	// Check for sequences above threshold
	for i, arr := range sequenceData {
		if allAboveThreshold(arr, threshold) && len(arr) >= arrSequenceLength {
			timestamp := timestampArr[i*arrSequenceLength]
			formattedTime := time.UnixMilli(timestamp).Format("2006-01-02 15:04:05")
			thresholds = append(thresholds, struct {
				Timestamp string
				Values    []float64
			}{Timestamp: formattedTime, Values: arr})
		}
	}

	log.Println("THRESHOLD", thresholds)
	return thresholds
}

func allAboveThreshold(arr []float64, threshold float64) bool {
	for _, num := range arr {
		if num <= threshold {
			return false
		}
	}
	return true
}

//func Run() {
//	//
//	sendTo := []string{"calebb.jnr@gmail.com"}
//	//
//	//	messages := MetricEngine()
//	//
//	//	// Construct the email subject
//	//	//subject := fmt.Sprintf("Alert: %d Threshold Messages for %s", len(groupMessages), group)
//	//	subject := fmt.Sprintf("Alert Threshold Messages")
//	//
//	actionURL := ""
//	//
//	err_ := messaging.SendEmail(sendTo, "Test Subject", "Hello World from Go")
//	if err_ != nil {
//		return
//	}
//
//	emailBody := messaging.FormatEmailMessageToSend("Hello World from Go", actionURL)
//	// Send the email
//	if err := messaging.SendEmail(sendTo, subject, emailBody); err != nil {
//		log.Printf("Failed to send email to %s: %v", group, err)
//	} else {
//		log.Printf("Alert sent to %s", group)
//	}
//
//	extraInfo := map[string]string{}
//
//	slackClient := messaging.SlackBotClient()
//	slackMessage := messaging.FormatSlackMessageToSend("Test Notification", "Hello World from Go", "critical", actionURL, extraInfo)
//
//	_, err := slackClient.SendSlackMessage("admin_x", slackMessage)
//	if err != nil {
//		return
//	}
//}

package monitors

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"sync"
	"time"

	"github.com/ZEGIFTED/MS.GoMonitor/internal"
	"github.com/ZEGIFTED/MS.GoMonitor/notifier"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/utils"
	"github.com/google/uuid"
	_ "github.com/microsoft/go-mssqldb"
)

func (sm *ServiceMonitor) LoadServicesFromDatabase() error {
	log.Println("Fetching Services...")

	rows, err := sm.Db.QueryContext(sm.Ctx, "EXEC ServiceReport @SERVICE_LEVEL = 'MONITOR', @VP = 1;")
	if err != nil {
		return fmt.Errorf("error querying services: %v", err)
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

		err := rows.Scan(
			&uuidStr,
			&service.Name,
			&service.Host,
			&service.Port,
			&service.VP,
			&service.Device,
			&service.FailureCount,
			&service.RetryCount,
			&Configuration,
			&service.CheckInterval,
			&service.IsAcknowledged,
			&service.SnoozeUntil,
			&service.AgentAPIBaseURL,
		)

		if err != nil {
			return fmt.Errorf("error scanning service row: %v", err)
		}

		service.SystemMonitorId, err = uuid.Parse(uuidStr)
		if err != nil {
			slog.Error(err.Error())
		}

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

	//for i, service := range services {
	//	fmt.Println(i, service)
	//}

	log.Printf("Loaded %d active Services", len(services))
	return nil
}

// StartService begins the monitoring process
func (sm *ServiceMonitor) StartService() error {
	log.Println("Starting Up MS Monitoring Service...")

	// Load initial services
	if err := sm.LoadServicesFromDatabase(); err != nil {
		log.Fatalf("Failed to load initial services: %v", err)
	}

	// Schedule service checks for each service
	for _, service := range sm.Services {
		serviceCopy := service
		interval := serviceCopy.CheckInterval

		if interval == "" || !utils.IsValidCron(interval) {
			interval = "*/1 * * * *"
		}

		_, err := sm.Cron.AddFunc(interval, func() {
			// Check if context is cancelled before running check
			if sm.Ctx.Err() != nil {
				return
			}
			sm.CheckService(serviceCopy)
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
func (sm *ServiceMonitor) StopService() {
	sm.Cancel() // Cancel context

	log.Printf("Stopping MS-SVC_MONITOR")
	close(sm.Alerts)

	// Stop the cron scheduler
	ctx := sm.Cron.Stop()

	// Wait for running jobs to complete (with timeout)
	select {
	case <-ctx.Done():
		log.Println("All jobs completed")
	case <-time.After(30 * time.Second):
		log.Println("Shutdown timed out waiting for jobs")
	}
}

// SendAlert sends an alert to a configured webhook
func (sm *ServiceMonitor) SendAlert(services []ServiceMonitorStatus) {
	// Implement alert sending logic (e.g., HTTP POST to webhook)

	//filePath := utils.GenerateServiceDowntimeAlert()

	//messaging.
	//slackClient := messaging.SlackBotClient()
	//slackMessage := messaging.FormatSlackMessageToSend("Test Notification", "Hello World from Go", "", "actionURL", extraInfo)
	//
	//_, err_ := slackClient.SendSlackMessage("admin_x", slackMessage)
	//if err_ != nil {
	//	return
	//}

	//sendTo := []string{"calebb.jnr@gmail.com", "cboluwade@nibss-plc.com.ng"}
	//messaging.SendReportEmail(sendTo, filePath)
}

// Example: Thread-safe iteration with RWMutex
//func (sm *ServiceMonitor) GetAllStatuses() []ServiceMonitorStatus {
//	sm.MU.RLock()
//	defer sm.MU.RUnlock()
//
//	var statuses []ServiceMonitorStatus
//	for _, status := range sm.StatusTracking {
//		statuses = append(statuses, *status)
//	}
//	return statuses
//}

// CheckService monitors a single service
func (sm *ServiceMonitor) CheckService(service ServiceMonitorData) {
	// Get the appropriate checker for this service type
	checker, exists := sm.Checkers[service.Device]

	if !exists {
		sm.handleUnknownServiceType(service)
		return
	}

	// Load or initialize service status atomically
	statusIface, _ := sm.StatusTracking.LoadOrStore(service.Name, &ServiceMonitorStatus{
		Name:          service.Name,
		Device:        service.Device,
		LiveCheckFlag: constants.UnknownStatus,
		LastCheckTime: time.Now(),
		FailureCount:  service.FailureCount,
	})
	currentStatus := statusIface.(*ServiceMonitorStatus)

	// Perform the service check
	message, isHealthy := checker.Check(service, sm.Ctx, sm.Db)
	//currentStatus.LastCheckTime = time.Now()

	//currentStatus, exists := sm.StatusTracking[service.Name]

	// Handle service health status
	if !isHealthy {
		sm.handleServiceFailure(service, currentStatus, message)
	} else {
		sm.handleServiceRecovery(service, currentStatus)
	}

	// Implement database logging logic to Update the database
	sm.updateDatabase(service, currentStatus)

	// Update the service status in sync.Map
	sm.StatusTracking.Store(service.Name, currentStatus)

	// Call the MetricEngine
	//monitors.MetricEngine()
}

func (sm *ServiceMonitor) handleUnknownServiceType(service ServiceMonitorData) {
	sm.MU.Lock()
	defer sm.MU.Unlock()

	currentStatus := &ServiceMonitorStatus{
		Name:          service.Name,
		Device:        service.Device,
		Status:        "Failed Monitor Check; No Monitor Impl found for Service type",
		LiveCheckFlag: constants.UnknownStatus,
		LastCheckTime: time.Now(),
	}

	// Update the service status in sync.Map
	//sm.StatusTracking[service.Name] = currentStatus
	sm.StatusTracking.Store(service.Name, currentStatus)

	// log.Printf("Service -> [%s]: No Checker Found For Type %s", service.Name, service.Device)
}

func (sm *ServiceMonitor) handleServiceFailure(service ServiceMonitorData, currentStatus *ServiceMonitorStatus, message ServiceMonitorStatus) {
	// Log service failure
	serviceAlertIdentifier := service.SystemMonitorId.String() + "|" + service.Name
	log.Printf("Service -> [%s] failed. Alert Identifier::: %s", service.Name, serviceAlertIdentifier)

	// Update status
	currentStatus.Status = message.Status
	currentStatus.FailureCount++

	// Determine escalation level
	if currentStatus.FailureCount > 3 {
		currentStatus.LiveCheckFlag = constants.Degraded
	} else if currentStatus.FailureCount > 1 {
		currentStatus.LiveCheckFlag = constants.Escalation
	}

	// Throttle alerts to avoid alert fatigue
	if lastAlert, ok := sm.AlertCache.Load(serviceAlertIdentifier); !ok || func() bool {
		lastAlertTime, valid := lastAlert.(time.Time)
		return valid && time.Since(lastAlertTime) < constants.AlertThrottleTime // Prevent alert spam within 10-min window
	}() {
		log.Println(serviceAlertIdentifier, currentStatus.FailureCount > constants.FailureThresholdCount, service.IsAcknowledged)
		if currentStatus.FailureCount > constants.FailureThresholdCount && !service.IsAcknowledged {
			// sm.Alerts <- internal.ServiceAlertEvent{
			// 	SystemMonitorId: service.SystemMonitorId,
			// 	ServiceName:     service.Name,
			// 	Message:         fmt.Sprintf("%s is down: %s", service.Name, message.Status),
			// 	Severity:        "critical",
			// 	Timestamp:       time.Now(),
			// 	AgentRepository: service.AgentRepository,
			// 	AgentAPI:        service.AgentAPIBaseURL,
			// }

			serviceAlertMessage := notifier.NotiferEvent{
				Title:      fmt.Sprintf("%s is down", service.Name),
				Identifier: serviceAlertIdentifier,
				Message:    message.Status,
				Timestamp:  time.Now().Format(time.RFC3339),
			}

			notifier.SendNotification(serviceAlertMessage)

			log.Printf("Added Failed Service to Alert Channel: %s", serviceAlertIdentifier)
			return
		}

		sm.AlertCache.Store(serviceAlertIdentifier, time.Now())
	}

	//if ok {
	//	if lastAlertTime, valid := lastAlert.(time.Time); valid {
	//		// Use lastAlertTime safely here
	//
	//		if !ok || time.Since(lastAlertTime) > constants.AlertThrottleTime {
	//			if currentStatus.FailureCount > constants.FailureThresholdCount && !service.IsAcknowledged {
	//				log.Printf("Adding Service to Alert Channel: %s", service.Name)
	//
	//				sm.Alerts <- internal.ServiceAlertEvent{
	//					ServiceName: service.Name,
	//					Message:     fmt.Sprintf("%s is down: %s", service.Name, message.LastErrorLog),
	//					Severity:    "critical",
	//					Timestamp:   time.Now(),
	//				}
	//			}
	//
	//			sm.AlertCache.Store(service.Name, time.Now())
	//		}
	//	} else {
	//		log.Println("Type assertion failed for lastAlert")
	//	}
	//} else {
	//	log.Println("No previous alert found for service:", service.Name)
	//}

	// Track service failure
	//sm.TrackServiceFailure(service, currentStatus, "")
}

func (sm *ServiceMonitor) handleServiceRecovery(service ServiceMonitorData, currentStatus *ServiceMonitorStatus) {
	// Update status for healthy service
	currentStatus.LiveCheckFlag = constants.Healthy
	currentStatus.Status = "Service is healthy."
	currentStatus.FailureCount = 0
	currentStatus.LastServiceUpTime = time.Now()

	// Log service recovery
	log.Printf("Service %s is healthy.", service.Name)
}

func (sm *ServiceMonitor) updateDatabase(service ServiceMonitorData, currentStatus *ServiceMonitorStatus) {
	// Update the database with the latest status
	_, err := sm.Db.Exec(`
        UPDATE [dbo].[SystemMonitor] 
        SET 
            Status = @Status, 
            LiveCheckFlag = @LiveCheckFlag, 
            LastServiceUpTime = @LastServiceUpTime, 
            LastCheckTime = @LastCheckTime, 
            FailureCount = @FailureCount
        WHERE 
            ServiceName = @Name`,
		sql.Named("Status", currentStatus.Status),
		sql.Named("LiveCheckFlag", currentStatus.LiveCheckFlag),
		sql.Named("LastServiceUpTime", currentStatus.LastServiceUpTime),
		sql.Named("LastCheckTime", currentStatus.LastCheckTime),
		sql.Named("FailureCount", currentStatus.FailureCount),
		sql.Named("Name", service.Name),
	)

	if err != nil {
		log.Printf("Error updating SystemMonitor for service %s: %v", service.Name, err)
	}
}

//
//// TrackServiceFailure logs the service failure
//func (sm *ServiceMonitor) TrackServiceFailure(service ServiceMonitorData, status *ServiceMonitorStatus, severity string) {
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

func (sm *ServiceMonitor) GetUnprocessedAlertsCount() int {
	sm.MU.RLock()         // Lock for read-only access
	defer sm.MU.RUnlock() // Unlock after function execution
	return len(sm.Alerts)
}

// AlertHandler checks if an alert should be sent (rate-limiting) and sends them.
func (sm *ServiceMonitor) AlertHandler() {
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
					agentStatsEndpoint, aErr := alert.AgentRepository.ValidateAgentURL(alert.AgentAPI, "/api/v1/agent/resource-usage?limit=5")

					if aErr != nil {
						log.Println(aErr.Error())
					}

					stats, statsErr := alert.AgentRepository.GetAgentServiceStats(agentStatsEndpoint)

					if statsErr != nil {
						log.Println(statsErr.Error())
					}

					log.Println(stats)
					alert.ServiceStats = stats
				}

				// Send the notification asynchronously
				go func(alert internal.ServiceAlertEvent, recipients internal.NotificationRecipients) {
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

func (sm *ServiceMonitor) SendDowntimeServiceNotification(event internal.ServiceAlertEvent, recipients internal.NotificationRecipients) error {
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

		go func(r []internal.NotificationRecipient) {
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

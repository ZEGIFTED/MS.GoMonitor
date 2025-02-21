package utils

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/microsoft/go-mssqldb"
	"time"
)

// StartService begins the monitoring process
func (sm *ServiceMonitor) StartService() error {
	// Load initial services
	if err := sm.LoadServicesFromDatabase(); err != nil {
		sm.Logger.Fatalf("Failed to load initial services: %v", err)
	}

	// Schedule checks for each service
	// Schedule service checks
	for _, service := range sm.Services {
		serviceCopy := service
		interval := serviceCopy.CheckInterval

		if interval == "" || !isValidCron(interval) {
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
			sm.Logger.Printf("Failed to schedule service %s: %v", serviceCopy.Name, err)
		}
	}

	sm.Cron.Start()
	return nil
}

// StopService begins the monitoring process
func (sm *ServiceMonitor) StopService() {
	sm.Cancel() // Cancel context

	// Stop the cron scheduler
	ctx := sm.Cron.Stop()

	// Wait for running jobs to complete (with timeout)
	select {
	case <-ctx.Done():
		sm.Logger.Println("All jobs completed")
	case <-time.After(30 * time.Second):
		sm.Logger.Println("Shutdown timed out waiting for jobs")
	}
}

// sendAlert sends an alert to a configured webhook
func (sm *ServiceMonitor) sendAlert(service ServiceMonitorConfig, status *ServiceMonitorStatus) {
	// Implement alert sending logic (e.g., HTTP POST to webhook)
	sm.Logger.Printf("ALERT: Service %s failed. Type: %s, Failure count: %d",
		service.Name, service.Device, status.FailureCount)
}

func (sm *ServiceMonitor) LoadServicesFromDatabase() error {
	query := `EXEC ServiceReport @SERVICE_LEVEL = 'MONITOR', @VP = 1;`

	//rows, err := sm.db.Query(query)
	rows, err := sm.Db.QueryContext(context.Background(), query)
	if err != nil {
		return fmt.Errorf("error querying services: %v", err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {

		}
	}(rows)

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
			sm.Logger.Printf("Warning: could not parse configuration for service %s: %v", service.Name, err)
			continue
		}

		sm.Logger.Println(service)
		services = append(services, service)
	}

	sm.MU.Lock()
	sm.Services = services
	sm.MU.Unlock()

	sm.Logger.Printf("Loaded %d active services", len(services))
	return nil
}

// CheckService monitors a single service
func (sm *ServiceMonitor) CheckService(service ServiceMonitorConfig) {
	sm.MU.Lock()
	status, exists := sm.StatusTracking[service.Name]
	if !exists {
		status = &ServiceMonitorStatus{
			Name:          service.Name,
			Device:        service.Device,
			LiveCheckFlag: UnknownStatus,
			LastCheckTime: time.Now(),
			FailureCount:  0,
		}
		sm.StatusTracking[service.Name] = status
	}
	sm.MU.Unlock()

	// Get the appropriate checker for this service type
	checker, ok := sm.Checkers[service.Device]

	if !ok {
		status.Status = "Failed Monitor Check; No Monitor Impl found for Service type"
		status.LiveCheckFlag = UnknownStatus
		//status.FailureCount++
		status.LastCheckTime = time.Now()
		status.LastErrorLog = "No Monitor Impl found for service type"

		sm.Logger.Printf("No checker found for service type: %s", service.Device)
		return
	}

	// Perform the service check
	isHealthy, message := checker.Check(service)

	sm.MU.Lock()
	defer sm.MU.Unlock()

	status.LastCheckTime = time.Now()

	if !isHealthy {
		// Log service failure
		fmt.Println(message.LastErrorLog)
		status.LastErrorLog = message.LastErrorLog
		status.Status = "Failed Monitor Check; " + message.LastErrorLog
		status.FailureCount++

		if status.FailureCount > 1 {
			status.LiveCheckFlag = Escalation
		} else if status.FailureCount > 3 {
			status.LiveCheckFlag = Degraded
		}

		if service.VP && !service.IsAcknowledged {
			sm.LogServiceFailure(service, status)

			// Send alert if configured
			sm.sendAlert(service, status)
		}

		sm.Logger.Printf("Service %s failed. Reason::: %s", service.Name, message.Status)
	} else {
		status.LiveCheckFlag = Healthy
		status.Status = "Service is healthy."
		status.FailureCount = 0
		status.LastErrorLog = ""
		status.LastServiceUpTime = time.Now()

		sm.Logger.Printf("Service %s is healthy. Message::: > %s", service.Name, message.Status)
	}

	// Implement database logging logic
	_, err := sm.Db.Exec(`
	UPDATE [dbo].[SystemMonitor] 
	SET 
		Status = @Status, 
		LiveCheckFlag = @LiveCheckFlag, 
		LastServiceUpTime = @LastServiceUpTime, 
		LastCheckTime = @LastCheckTime, 
		FailureCount = @FailureCount
		-- RetryCount = @RetryCount 
	WHERE 
		ServiceName = @Name`,
		sql.Named("Status", status.Status),
		sql.Named("LiveCheckFlag", status.LiveCheckFlag),
		sql.Named("LastServiceUpTime", status.LastServiceUpTime),
		sql.Named("LastCheckTime", status.LastCheckTime),
		sql.Named("FailureCount", status.FailureCount),
		// sql.Named("RetryCount", status.RetryCount),
		sql.Named("Name", service.Name),
	)

	if err != nil {
		sm.Logger.Printf("Error updating SystemMonitor: %v", err)
	}

	// Call the MetricEngine
	//monitors.MetricEngine()
}

// LogServiceFailure logs the service failure
func (sm *ServiceMonitor) LogServiceFailure(service ServiceMonitorConfig, status *ServiceMonitorStatus) {

	sm.Logger.Printf("Service Failure - %s -> Name: %s, Type: %s, Error: %s",
		service.Id, service.Name, service.Device, status.LastErrorLog)

}

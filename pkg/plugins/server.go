package plugins

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"log/slog"
	"time"

	"github.com/ZEGIFTED/MS.GoMonitor/monitors"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
	mstypes "github.com/ZEGIFTED/MS.GoMonitor/types"
)

type ServerMonitorPlugin struct {
	config map[string]any
}

func NewServerMonitorPlugin() *ServerMonitorPlugin {
	return &ServerMonitorPlugin{}
}

func (p *ServerMonitorPlugin) Initialize(config map[string]interface{}) error {
	p.config = config
	return nil
}

func (s *ServerMonitorPlugin) Check(ctx context.Context, db *sql.DB, server monitors.ServiceMonitorData) (monitors.MonitoringResult, error) {
	status := monitors.MonitoringResult{
		SystemMonitorId: server.SystemMonitorId.String(),
		ServicePluginID: s.Name(),
		HealthReport:    constants.GetStatusInfo(constants.UnknownStatus, ""),
		LastCheckTime:   time.Now(),
	}

	query := `
		SELECT [AgentID], [Timestamp], [CPUUsage], [TimestampMem], [MemoryUsage]
		FROM SystemMetricData
		`

	// Execute the query
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			status.FailureCount++
			status.HealthReport = constants.GetStatusInfo(constants.Acknowledged, "No Server Metrics "+err.Error())
			return status, err
		}

		log.Println("Server Check Error: ", err.Error())

		status.FailureCount++
		status.HealthReport = constants.GetStatusInfo(constants.Escalation, "Error Retrieving Server Metrics "+err.Error())
		return status, err
	}

	if rows != nil {
		defer func(rows *sql.Rows) {
			err__ := rows.Close()
			if err__ != nil {
				log.Println(err__.Error())
			}
		}(rows)
	}

	agentMetrics := make(map[string][]mstypes.Metric)

	for rows.Next() {
		var metric mstypes.Metric
		err_ := rows.Scan(
			&metric.AgentID,
			&metric.Timestamp,
			&metric.CPUUsage,
			&metric.TimestampMem,
			&metric.MemoryUsage,
		)

		if err_ != nil {
			slog.ErrorContext(ctx, "Error scanning row:", "Error", err_.Error())
			continue
		}

		agentMetrics[metric.AgentID] = append(agentMetrics[metric.AgentID], metric)
	}

	if server.Device == "Docker" {
		agentHttpClient, nt, err := server.AgentRepository.ValidateAgentURL(server.AgentAPIBaseURL, "api/v1/agent/container")
		if err != nil {
			status.FailureCount++
			status.HealthReport = constants.GetStatusInfo(constants.Escalation, err.Error())
			return status, err
		}

		_, tErr := server.AgentRepository.GetAgentContainerStats(agentHttpClient, nt)
		if tErr != nil {
			slog.Error(" fetching agent thresholds:", "Error", tErr.Error())
		}
	}

	agentHttpClient, agentThresholdEndpoint, err := server.AgentRepository.ValidateAgentURL(server.AgentAPIBaseURL, "api/v1/agent/config")
	if err != nil {
		status.FailureCount++
		status.HealthReport = constants.GetStatusInfo(constants.Escalation, err.Error())
		return status, err
	}

	//log.Println(agentThresholdEndpoint, len(agentMetrics), agentMetrics)

	for agentID, metrics := range agentMetrics {
		agentThresholds, tErr := server.AgentRepository.GetAgentThresholds(agentHttpClient, agentThresholdEndpoint)
		if tErr != nil {
			slog.Error(" fetching agent thresholds:", "Error", tErr.Error())
			continue
		}

		// Placeholder for metric processing
		slog.Info("Processing metrics", "AgentId", agentID)
		MetricEngine(agentThresholds, metrics)
	}

	// return monitors.ServiceMonitorStatus{
	// 	Name:          server.Name,
	// 	Device:        server.Device,
	// 	LiveCheckFlag: constants.UnknownStatus,
	// 	Status:        "Configuration Setup In Progress",
	// 	LastCheckTime: time.Now(),
	// 	FailureCount:  1,
	// }, fmt.Errorf("Configuration Setup In Progress")

	status.FailureCount = 0
	status.HealthReport = constants.GetStatusInfo(constants.Healthy, "")

	return status, nil
}

func (p *ServerMonitorPlugin) Name() string {
	return "http_monitor"
}

func (p *ServerMonitorPlugin) Description() string {
	return "HTTP service monitor"
}

func (p *ServerMonitorPlugin) SupportedTypes() []monitors.ServiceType {
	return []monitors.ServiceType{monitors.ServiceMonitorWebModules, monitors.ServiceMonitorServer}
}

func (hc *ServerMonitorPlugin) Cleanup() error {
	return nil
}

var ServerPlugin monitors.ServiceMonitorPlugin = NewServerMonitorPlugin()

// MetricEngine Aggregates all metric sources by AppId and AgentId
func MetricEngine(agentThresholds mstypes.AgentThresholdResponse, metrics []mstypes.Metric) {
	var cpuTSdata []monitors.TimeSeriesData
	for _, metric := range metrics {
		cpuTSdata = append(cpuTSdata, monitors.TimeSeriesData{
			Timestamp: metric.Timestamp,
			Value:     metric.CPUUsage,
		})
	}

	var memTSdata []monitors.TimeSeriesData
	for _, metric := range metrics {
		memTSdata = append(memTSdata, monitors.TimeSeriesData{
			Timestamp: metric.TimestampMem,
			Value:     metric.CPUUsage,
		})
	}

	// get Agent use

	monitors.CheckTSDataAboveThreshold(cpuTSdata, 80, 5)
	monitors.CheckTSDataAboveThreshold(memTSdata, 80, 5)
}

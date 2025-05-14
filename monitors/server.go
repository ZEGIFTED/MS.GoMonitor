package monitors

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"log/slog"
	"time"

	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
	mstypes "github.com/ZEGIFTED/MS.GoMonitor/types"
)

func (service *ServerHealthChecker) Check(server ServiceMonitorData, ctx context.Context, db *sql.DB) (ServiceMonitorStatus, bool) {
	query := `
		SELECT [AgentID], [Timestamp], [CPUUsage], [TimestampMem], [MemoryUsage]
		FROM SystemMetricData
		`

	// Execute the query
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ServiceMonitorStatus{
				Status: "no data found in the database",
			}, false
		}

		log.Println("Server Check Error: ", err.Error())

		return ServiceMonitorStatus{
			Status: "Server Check Database Error",
		}, false
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
			return ServiceMonitorStatus{
				Status: err.Error(),
			}, false
		}

		_, tErr := server.AgentRepository.GetAgentContainerStats(agentHttpClient, nt)
		if tErr != nil {
			slog.Error(" fetching agent thresholds:", "Error", tErr.Error())
		}
	}

	agentHttpClient, agentThresholdEndpoint, err := server.AgentRepository.ValidateAgentURL(server.AgentAPIBaseURL, "api/v1/agent/config")
	if err != nil {
		return ServiceMonitorStatus{
			Status: err.Error(),
		}, false
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
		service.MetricEngine(agentThresholds, metrics)
	}

	return ServiceMonitorStatus{
		Name:          server.Name,
		Device:        server.Device,
		LiveCheckFlag: constants.UnknownStatus,
		Status:        "Configuration Setup In Progress",
		LastCheckTime: time.Now(),
		FailureCount:  1,
	}, false
}

// MetricEngine Aggregates all metric sources by AppId and AgentId
func (service *ServerHealthChecker) MetricEngine(agentThresholds mstypes.AgentThresholdResponse, metrics []mstypes.Metric) {
	var cpuTSdata []TimeSeriesData
	for _, metric := range metrics {
		cpuTSdata = append(cpuTSdata, TimeSeriesData{
			Timestamp: metric.Timestamp,
			Value:     metric.CPUUsage,
		})
	}

	var memTSdata []TimeSeriesData
	for _, metric := range metrics {
		memTSdata = append(memTSdata, TimeSeriesData{
			Timestamp: metric.TimestampMem,
			Value:     metric.CPUUsage,
		})
	}

	// get Agent use

	CheckTSDataAboveThreshold(cpuTSdata, 80, 5)
	CheckTSDataAboveThreshold(memTSdata, 80, 5)
}

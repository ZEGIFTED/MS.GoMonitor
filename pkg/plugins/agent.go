package plugins

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/ZEGIFTED/MS.GoMonitor/internal/repository"
	"github.com/ZEGIFTED/MS.GoMonitor/monitors"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/utils"
	mstypes "github.com/ZEGIFTED/MS.GoMonitor/types"
)

type AgentMonitorPlugin struct {
	config map[string]interface{}
}

func NewAgentMonitorPlugin() *AgentMonitorPlugin {
	return &AgentMonitorPlugin{}
}

func (p *AgentMonitorPlugin) Initialize(config map[string]interface{}) error {
	p.config = config
	return nil
}

func (a *AgentMonitorPlugin) Check(_ context.Context, _ *sql.DB, agent monitors.ServiceMonitorData) (monitors.MonitoringResult, error) {
	agentHttpClient, agentAddress, err := agent.AgentRepository.ValidateAgentURL(agent.AgentAPIBaseURL, "/api/v1/agent/health")

	status := monitors.MonitoringResult{
		SystemMonitorId: agent.SystemMonitorId.String(),
		ServicePluginID: a.Name(),
		HealthReport:    constants.GetStatusInfo(constants.UnknownStatus, ""),
		LastCheckTime:   time.Now(),
	}

	if err != nil {
		status.FailureCount++
		status.HealthReport = constants.GetStatusInfo(constants.InvalidConfiguration, "Invalid URL configuration "+err.Error())

		return status, err
	}

	slog.Info("Calling Agent API", "Endpoint", agentAddress)
	resp, err := agentHttpClient.Get(agentAddress)

	if err != nil {
		status.FailureCount++
		status.HealthReport = constants.GetStatusInfo(constants.Degraded, "Invalid URL configuration "+err.Error())

		return status, err
	}

	defer func(Body io.ReadCloser) {
		__err := Body.Close()
		if __err != nil {
			slog.Error("Error closing response body: %v", "Ex", __err.Error())
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		status.FailureCount++
		status.HealthReport = constants.GetStatusInfo(constants.Escalation, "Unable to Sync Agent Metrics "+err.Error())

		return status, err
	}

	if resp.StatusCode != http.StatusOK {
		status.FailureCount++
		status.HealthReport = constants.GetStatusInfo(constants.Escalation, fmt.Sprintf("Unsuccessful Agent Call: %d", resp.StatusCode))

		return status, err
	}

	var apiResponse mstypes.AgentMetricResponse
	err = json.Unmarshal(body, &apiResponse)

	//log.Println("Agent API response", apiResponse, err)

	if err != nil {
		status.FailureCount++
		status.HealthReport = constants.GetStatusInfo(constants.Escalation, "Unable to Sync Agent Metrics. Failed To Decode Response")

		return status, err
	}

	// Convert data to our Agent struct
	agentData := mstypes.AgentInfo{
		AgentID:    apiResponse.AgentInfo.AgentID,
		Name:       apiResponse.AgentInfo.Name,
		IPAddress:  apiResponse.AgentInfo.IPAddress,
		OS:         apiResponse.AgentInfo.OS,
		Version:    apiResponse.AgentInfo.Version,
		SDKVersion: apiResponse.AgentInfo.SDKVersion,
	}

	// Convert CPU & Memory metrics
	for i := 0; i < len(apiResponse.SystemInfo.CPU) && i < len(apiResponse.SystemInfo.Memory); i++ {
		//for i, pair := range apiResponse.SystemInfo.CPU {
		//	agent.Metrics[i] = Metric{
		//		Timestamp:   int64(pair[0]),
		//		CPUUsage:    pair[1],
		//		MemoryUsage: apiResponse.SystemInfo.Memory[i][1],
		//	}
		//}

		// Append metric with properly converted types'
		agentData.Metrics = append(agentData.Metrics, mstypes.Metric{
			Timestamp:    int64(apiResponse.SystemInfo.CPU[i][0]), // Convert to int64
			TimestampMem: int64(apiResponse.SystemInfo.Memory[i][0]),
			CPUUsage:     apiResponse.SystemInfo.CPU[i][1],
			MemoryUsage:  apiResponse.SystemInfo.Memory[i][1],
		})
	}

	// Convert Disk metrics
	for _, disk := range apiResponse.SystemInfo.Disk {
		agentData.Disks = append(agentData.Disks, mstypes.DiskMetric{
			Drive:      disk.Drive,
			Size:       disk.Size,
			Free:       disk.Free,
			Used:       disk.Used,
			FormatSize: fmt.Sprintf("%.2f GB", float64(disk.Size)/1e9),
			FormatFree: fmt.Sprintf("%.2f GB", float64(disk.Free)/1e9),
		})
	}

	var agents []mstypes.AgentInfo

	//service.LastCheckTime = service.LastCheckTime.Add(1 * time.Minute)
	//check.LastCheckTime = time.Now()
	//service.Check()
	//log.Println("Agent API", apiResponse.AgentInfo.AgentID, "code", agent)
	agents = append(agents, agentData)

	if len(agents) > 0 {
		db := utils.DatabaseConnection()

		agentHttpClient, agentSyncURL, err_ := agent.AgentRepository.ValidateAgentURL(agent.AgentAPIBaseURL, "/api/v1/agent/sync_complete")

		if err_ != nil {
			status.FailureCount++
			status.HealthReport = constants.GetStatusInfo(constants.Degraded, fmt.Sprintf("Unable to Sync Agent Metrics. %s", err_.Error()))

			return status, err
		}

		if syncErr := repository.SyncAgentMetrics(db, agents, agentHttpClient, agentSyncURL); syncErr != nil {
			status.FailureCount++
			status.HealthReport = constants.GetStatusInfo(constants.Escalation, fmt.Sprintf("Unable to Sync Agent Metrics. %s", syncErr.Error()))

			return status, err
		}
	}

	status.FailureCount = 0
	status.HealthReport = constants.GetStatusInfo(constants.Healthy, "")

	return status, nil
}

func (p *AgentMonitorPlugin) Name() string {
	return "agent_monitor"
}

func (p *AgentMonitorPlugin) Description() string {
	return "Agent service monitor"
}

func (p *AgentMonitorPlugin) SupportedTypes() []monitors.ServiceType {
	return []monitors.ServiceType{monitors.ServiceMonitorWebModules, monitors.ServiceMonitorServer}
}

func (hc *AgentMonitorPlugin) Cleanup() error {
	return nil
}

var AgentPlugin monitors.ServiceMonitorPlugin = NewAgentMonitorPlugin()

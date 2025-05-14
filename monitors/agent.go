package monitors

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
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/utils"
	mstypes "github.com/ZEGIFTED/MS.GoMonitor/types"
)

func (service *AgentServiceChecker) Check(agent ServiceMonitorData, _ context.Context, _ *sql.DB) (ServiceMonitorStatus, bool) {
	agentHttpClient, agentAddress, err := agent.AgentRepository.ValidateAgentURL(agent.AgentAPIBaseURL, "/api/v1/agent/health")

	if err != nil {
		return ServiceMonitorStatus{
			Name:          agent.Name,
			Device:        agent.Device,
			LiveCheckFlag: constants.Degraded,
			Status:        "Invalid URL configuration " + err.Error(),
			LastCheckTime: time.Now(),
			FailureCount:  0,
		}, false
	}

	slog.Info("Calling Agent API", "Endpoint", agentAddress)
	resp, err := agentHttpClient.Get(agentAddress)

	if err != nil {
		return ServiceMonitorStatus{
			Name:          agent.Name,
			Device:        agent.Device,
			LiveCheckFlag: constants.Degraded,
			Status:        "Agent Status Unknown " + err.Error(),
			LastCheckTime: time.Now(),
			FailureCount:  1,
		}, false
	}

	defer func(Body io.ReadCloser) {
		__err := Body.Close()
		if __err != nil {
			slog.Error("Error closing response body: %v", "Ex", __err.Error())
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return ServiceMonitorStatus{
			Name:          agent.Name,
			Device:        agent.Device,
			LiveCheckFlag: constants.Escalation,
			Status:        "Unable to Sync Agent Metrics " + err.Error(),
			LastCheckTime: time.Now(),
			FailureCount:  1,
		}, false
	}

	if resp.StatusCode != http.StatusOK {
		return ServiceMonitorStatus{
			Name:          agent.Name,
			Device:        agent.Device,
			LiveCheckFlag: constants.Escalation,
			Status:        fmt.Sprintf("Unsuccessful Agent Call: %d", resp.StatusCode),
			LastCheckTime: time.Now(),
			FailureCount:  1,
		}, false
	}

	var apiResponse mstypes.AgentMetricResponse
	err = json.Unmarshal(body, &apiResponse)

	//log.Println("Agent API response", apiResponse, err)

	if err != nil {
		return ServiceMonitorStatus{
			Name:          agent.Name,
			Device:        agent.Device,
			LiveCheckFlag: constants.Escalation,
			Status:        "Unable to Sync Agent Metrics",
			LastCheckTime: time.Now(),
			FailureCount:  1,
		}, false
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
			return ServiceMonitorStatus{
				Name:          agent.Name,
				Device:        agent.Device,
				LiveCheckFlag: constants.Degraded,
				Status:        err_.Error(),
				LastCheckTime: time.Now(),
				FailureCount:  1,
			}, false
		}

		if syncErr := repository.SyncAgentMetrics(db, agents, agentHttpClient, agentSyncURL); syncErr != nil {
			return ServiceMonitorStatus{
				Name:          agent.Name,
				Device:        agent.Device,
				Status:        "Error while syncing metrics " + syncErr.Error(),
				LiveCheckFlag: constants.Escalation,
				LastCheckTime: time.Now(),
				FailureCount:  1,
			}, false
		}
	}

	return ServiceMonitorStatus{
		Name:              agent.Name,
		Device:            agent.Device,
		LiveCheckFlag:     constants.Healthy,
		Status:            "Healthy",
		LastCheckTime:     time.Now(),
		LastServiceUpTime: time.Now(),
		FailureCount:      0,
	}, true
}

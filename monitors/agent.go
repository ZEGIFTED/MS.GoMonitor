package monitors

import (
	"encoding/json"
	"fmt"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/utils"
	"io"
	"log"
	"net/http"
	"time"
)

func (service *AgentServiceChecker) Check(config ServiceMonitorConfig) (bool, ServiceMonitorStatus) {
	// Get the URL from configuration
	host := config.Host
	protocol, ok := config.Configuration["protocol"]

	if !ok || (protocol != "https" && protocol != "http") {
		log.Println("invalid agent protocol in configuration... Using default")

		protocol = "http"
	}

	protocol = protocol.(string)

	if host == "" {
		return false, ServiceMonitorStatus{
			Name:          config.Name,
			Device:        config.Device,
			LiveCheckFlag: constants.Degraded,
			Status:        "Unknown",
			LastCheckTime: time.Now(),
			FailureCount:  0,
			LastErrorLog:  "Invalid URL configuration",
		}
	}

	port := config.Port
	agentAddress := fmt.Sprintf("%v://%s:%d/api/v1/agent/health", protocol, host, port)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	log.Println("Calling Agent API", agentAddress)
	resp, err := client.Get(agentAddress)

	if err != nil {
		return false, ServiceMonitorStatus{
			Name:          config.Name,
			Device:        config.Device,
			LiveCheckFlag: constants.Degraded,
			Status:        "Agent Status Unknown",
			LastCheckTime: time.Now(),
			FailureCount:  0,
			LastErrorLog:  fmt.Sprintf("HTTP check failed: %v", err),
		}
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return false, ServiceMonitorStatus{
			Name:          config.Name,
			Device:        config.Device,
			LiveCheckFlag: constants.Escalation,
			Status:        "OK",
			LastCheckTime: time.Now(),
			//LastServiceUpTime: time.Now(),
			FailureCount: 0,
			LastErrorLog: fmt.Sprintf("Unable to read HTTP Content: %d. %s", resp.StatusCode, err),
		}
	}

	if resp.StatusCode != http.StatusOK {
		return false, ServiceMonitorStatus{
			Name:          config.Name,
			Device:        config.Device,
			LiveCheckFlag: constants.Escalation,
			Status:        "OK",
			LastCheckTime: time.Now(),
			//LastServiceUpTime: time.Now(),
			FailureCount: 0,
			LastErrorLog: fmt.Sprintf("Unsuccessful Agent Call: %d. %s", resp.StatusCode, string(body)),
		}
	}

	var apiResponse AgentMetricResponse
	err = json.Unmarshal(body, &apiResponse)

	//log.Println("Agent API response", apiResponse, err)

	if err != nil {
		return false, ServiceMonitorStatus{
			Name:          config.Name,
			Device:        config.Device,
			LiveCheckFlag: constants.Escalation,
			Status:        "Unable to Sync Agent Metrics",
			LastCheckTime: time.Now(),
			//LastServiceUpTime: time.Now(),
			FailureCount: 0,
			LastErrorLog: fmt.Sprintf("Error parsing Agent JSON response: %d. %s", resp.StatusCode, err),
		}
	}

	// Convert data to our Agent struct
	agent := AgentInfo{
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
		agent.Metrics = append(agent.Metrics, Metric{
			Timestamp:    int64(apiResponse.SystemInfo.CPU[i][0]), // Convert to int64
			TimestampMem: int64(apiResponse.SystemInfo.Memory[i][0]),
			CPUUsage:     apiResponse.SystemInfo.CPU[i][1],
			MemoryUsage:  apiResponse.SystemInfo.Memory[i][1],
		})
	}

	// Convert Disk metrics
	for _, disk := range apiResponse.SystemInfo.Disk {
		agent.Disks = append(agent.Disks, DiskMetric{
			Drive:      disk.Drive,
			Size:       disk.Size,
			Free:       disk.Free,
			Used:       disk.Used,
			FormatSize: fmt.Sprintf("%.2f GB", float64(disk.Size)/1e9),
			FormatFree: fmt.Sprintf("%.2f GB", float64(disk.Free)/1e9),
		})
	}

	var agents []AgentInfo

	//service.LastCheckTime = service.LastCheckTime.Add(1 * time.Minute)
	//check.LastCheckTime = time.Now()
	//service.Check()
	//log.Println("Agent API", apiResponse.AgentInfo.AgentID, "code", agent)
	agents = append(agents, agent)

	if len(agents) > 0 {
		db := utils.DatabaseConnection()

		agentSyncURL := fmt.Sprintf("%v://%s:%d/api/v1/agent/sync_complete", protocol, host, port)
		err := SyncMetrics(db, agents, agentSyncURL)

		if err != nil {
			return false, ServiceMonitorStatus{
				Name:          config.Name,
				Device:        config.Device,
				Status:        "Error while syncing metrics " + err.Error(),
				LiveCheckFlag: constants.Escalation,
				LastCheckTime: time.Now(),
				FailureCount:  1,
			}
		}
	}

	return true, ServiceMonitorStatus{
		Name:              config.Name,
		Device:            config.Device,
		LiveCheckFlag:     constants.Healthy,
		Status:            "OK",
		LastCheckTime:     time.Now(),
		LastServiceUpTime: time.Now(),
		FailureCount:      0,
		LastErrorLog:      "",
	}
}

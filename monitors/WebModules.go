package monitors

import (
	"crypto/tls"
	"fmt"
	"github.com/ZEGIFTED/MS.GoMonitor/utils"
	"io"
	"log"
	"net/http"
	"time"
)

func (service *WebModulesServiceChecker) Check(config utils.ServiceMonitorConfig) (bool, utils.ServiceMonitorStatus) {
	// Create a custom HTTP client with disabled SSL verification
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	host := config.Host
	port := config.Port
	protocol, ok := config.Configuration["protocol"]

	if !ok || (protocol != "https" && protocol != "http") {
		log.Println("invalid agent protocol in configuration... Using default")

		protocol = "http"
	}

	protocol = protocol.(string)

	if host == "" {
		return false, utils.ServiceMonitorStatus{
			Id:            0,
			Name:          config.Name,
			Device:        config.Device,
			LiveCheckFlag: utils.Degraded,
			Status:        "Unknown",
			LastCheckTime: time.Now(),
			FailureCount:  0,
			LastErrorLog:  "Invalid URL configuration",
		}
	}

	webURL := fmt.Sprintf("%v://%s:%d", protocol, host, port)
	resp, err := httpClient.Get(webURL)

	if err != nil {
		return false, utils.ServiceMonitorStatus{
			//Id:            0,
			Name:          config.Name,
			Device:        config.Device,
			LiveCheckFlag: utils.Degraded,
			Status:        "Agent Status Unknown",
			LastCheckTime: time.Now(),
			FailureCount:  0,
			LastErrorLog:  "HTTP check failed: " + err.Error(),
		}
	}

	bodyBytes, err := io.ReadAll(resp.Body)

	if err != nil {
		return false, utils.ServiceMonitorStatus{
			Id:            0,
			Name:          config.Name,
			Device:        config.Device,
			LiveCheckFlag: utils.Escalation,
			Status:        "OK",
			LastCheckTime: time.Now(),
			//LastServiceUpTime: time.Now(),
			FailureCount: 0,
			LastErrorLog: fmt.Sprintf("Bd HTTP Status: %d. %s", resp.StatusCode, err),
		}
	}

	if resp.StatusCode != http.StatusOK {
		return false, utils.ServiceMonitorStatus{
			Id:            0,
			Name:          config.Name,
			Device:        config.Device,
			LiveCheckFlag: utils.Escalation,
			Status:        "OK",
			LastCheckTime: time.Now(),
			//LastServiceUpTime: time.Now(),
			FailureCount: 0,
			LastErrorLog: fmt.Sprintf("Bd HTTP Status: %d. %s", resp.StatusCode, string(bodyBytes)),
		}
	}

	return true, utils.ServiceMonitorStatus{
		Id:                0,
		Name:              config.Name,
		Device:            config.Device,
		LiveCheckFlag:     utils.Healthy,
		Status:            "OK",
		LastCheckTime:     time.Now(),
		LastServiceUpTime: time.Now(),
		FailureCount:      0,
		LastErrorLog:      "",
	}
}

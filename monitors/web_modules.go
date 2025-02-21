package monitors

import (
	"crypto/tls"
	"fmt"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
	"io"
	"log"
	"net/http"
	"time"
)

func (service *WebModulesServiceChecker) Check(config ServiceMonitorConfig) (bool, ServiceMonitorStatus) {
	// Create a custom HTTP client with disabled SSL verification
	httpClient := &http.Client{
		Timeout: constants.HTTPRequestTimeout,
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

	webURL := fmt.Sprintf("%v://%s:%d", protocol, host, port)
	resp, err := httpClient.Get(webURL)

	if err != nil {
		return false, ServiceMonitorStatus{
			Name:          config.Name,
			Device:        config.Device,
			LiveCheckFlag: constants.Degraded,
			Status:        "Agent Status Unknown",
			LastCheckTime: time.Now(),
			FailureCount:  0,
			LastErrorLog:  "HTTP check failed: " + err.Error(),
		}
	}

	bodyBytes, err := io.ReadAll(resp.Body)

	if err != nil {
		return false, ServiceMonitorStatus{
			Name:          config.Name,
			Device:        config.Device,
			LiveCheckFlag: constants.Escalation,
			Status:        "OK",
			LastCheckTime: time.Now(),
			//LastServiceUpTime: time.Now(),
			FailureCount: 0,
			LastErrorLog: fmt.Sprintf("Bd HTTP Status: %d. %s", resp.StatusCode, err),
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
			LastErrorLog: fmt.Sprintf("Bd HTTP Status: %d. %s", resp.StatusCode, string(bodyBytes)),
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

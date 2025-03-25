package monitors

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
)

func (service *WebModulesServiceChecker) Check(web ServiceMonitorData, _ context.Context, _ *sql.DB) (ServiceMonitorStatus, bool) {
	// Create a custom HTTP client with disabled SSL verification
	httpClient := &http.Client{
		Timeout: constants.HTTPRequestTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	host := web.Host
	port := web.Port
	protocol, ok := web.Configuration["protocol"]

	if !ok || (protocol != "https" && protocol != "http") {
		//log.Println("invalid agent protocol in configuration... Using default")

		protocol = "http"
	}

	protocol = protocol.(string)

	if host == "" {
		return ServiceMonitorStatus{
			Name:          web.Name,
			Device:        web.Device,
			LiveCheckFlag: constants.Degraded,
			Status:        "Invalid URL Configuration Setup",
			LastCheckTime: time.Now(),
			FailureCount:  1,
		}, false
	}

	webURL := fmt.Sprintf("%v://%s:%d", protocol, host, port)
	resp, err := httpClient.Get(webURL)

	if err != nil {
		return ServiceMonitorStatus{
			Name:          web.Name,
			Device:        web.Device,
			LiveCheckFlag: constants.Degraded,
			Status:        err.Error(),
			LastCheckTime: time.Now(),
			FailureCount:  1,
		}, false
	}

	//bodyBytes, err := io.ReadAll(resp.Body)

	//if err != nil {
	//	return ServiceMonitorStatus{
	//		Name:          config.Name,
	//		Device:        config.Device,
	//		LiveCheckFlag: constants.Escalation,
	//		Status:        fmt.Sprintf("Bd HTTP Status: %d. %s", resp.StatusCode, err),
	//		LastCheckTime: time.Now(),
	//		FailureCount:  0,
	//	}, false
	//}

	if resp.StatusCode != http.StatusOK {
		return ServiceMonitorStatus{
			Name:          web.Name,
			Device:        web.Device,
			LiveCheckFlag: constants.Escalation,
			Status:        fmt.Sprintf("Bad HTTP Status: %d.", resp.StatusCode),
			LastCheckTime: time.Now(),
			FailureCount:  1,
		}, false
	}

	return ServiceMonitorStatus{
		Name:              web.Name,
		Device:            web.Device,
		LiveCheckFlag:     constants.Healthy,
		Status:            "Healthy",
		LastCheckTime:     time.Now(),
		LastServiceUpTime: time.Now(),
		FailureCount:      0,
	}, true
}

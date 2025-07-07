package plugins

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/ZEGIFTED/MS.GoMonitor/monitors"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
)

type WebMonitorPlugin struct {
	config map[string]any
}

func NewWebMonitorPlugin() *WebMonitorPlugin {
	return &WebMonitorPlugin{}
}

func (p *WebMonitorPlugin) Initialize(config map[string]interface{}) error {
	p.config = config
	return nil
}

func (w *WebMonitorPlugin) Check(_ context.Context, _ *sql.DB, web monitors.ServiceMonitorData) (monitors.MonitoringResult, error) {
	status := monitors.MonitoringResult{
		SystemMonitorId: web.SystemMonitorId.String(),
		ServicePluginID: w.Name(),
		HealthReport:    constants.GetStatusInfo(constants.UnknownStatus, ""),
		LastCheckTime:   time.Now(),
	}

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

	if port == 0 {
		port = 80
	}

	protocol = protocol.(string)

	if host == "" {
		status.FailureCount++
		status.HealthReport = constants.GetStatusInfo(constants.InvalidConfiguration, "Invalid URL Configuration Setup")
		return status, fmt.Errorf("invalid URL: %v", "Invalid URL Configuration Setup")
	}

	webURL := fmt.Sprintf("%v://%s:%d", protocol, host, port)
	resp, err := httpClient.Get(webURL)

	if err != nil {
		status.FailureCount++
		status.HealthReport = constants.GetStatusInfo(constants.Escalation, err.Error())
		return status, err
	}

	if resp.StatusCode != http.StatusOK {
		status.FailureCount++
		status.HealthReport = constants.GetStatusInfo(constants.Degraded, fmt.Sprintf("Bad HTTP Status: %d", resp.StatusCode))
		return status, err
	}

	status.FailureCount = 0
	status.HealthReport = constants.GetStatusInfo(constants.Healthy, "")

	return status, nil
}

func (p *WebMonitorPlugin) Name() string {
	return "web_monitor"
}

func (p *WebMonitorPlugin) Description() string {
	return "Agent service monitor"
}

func (p *WebMonitorPlugin) SupportedTypes() []monitors.ServiceType {
	return []monitors.ServiceType{monitors.ServiceMonitorWebModules, monitors.ServiceMonitorServer}
}

func (hc *WebMonitorPlugin) Cleanup() error {
	return nil
}

var WebPlugin monitors.ServiceMonitorPlugin = NewAgentMonitorPlugin()

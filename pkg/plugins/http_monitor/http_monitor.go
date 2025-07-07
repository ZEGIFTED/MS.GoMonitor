package http_monitor

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ZEGIFTED/MS.GoMonitor/monitors"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
)

type HTTPMonitorPlugin struct {
	// name   string
	// url    string
	client  *http.Client
	config  map[string]any
	timeout time.Duration
}

func NewHTTPMonitorPlugin() *HTTPMonitorPlugin {
	return &HTTPMonitorPlugin{
		// name: name,
		// url:  url,
		// client: &http.Client{
		// 	Timeout: 10 * time.Second,
		// },
	}
}

func (h *HTTPMonitorPlugin) Initialize(config map[string]any) error {
	log.Println("Initializing HTTP/TCP/Telnet Monitor Plugin")
	h.config = config

	timeoutSec, ok := config["timeout"].(float64)
	if !ok {
		timeoutSec = 30 // default
	}
	h.timeout = time.Duration(timeoutSec) * time.Second
	// if timeout, ok := config["timeout"]; ok {
	// 	if timeoutStr, ok := timeout.(string); ok {
	// 		if d, err := time.ParseDuration(timeoutStr); err == nil {
	// 			h.client.Timeout = d
	// 		}
	// 	}
	// }
	return nil
}

func (p *HTTPMonitorPlugin) formatHTTPAddress(host string, port int) string {
	// Check if the host is an IPv6 address
	if strings.Count(host, ":") >= 2 && !strings.Contains(host, "%") {
		return fmt.Sprintf("http://[%s]:%d", host, port)
	}
	return fmt.Sprintf("http://%s:%d", host, port)
}

func (p *HTTPMonitorPlugin) Check(ctx context.Context, db *sql.DB, service monitors.ServiceMonitorData) (monitors.MonitoringResult, error) {
	status := monitors.MonitoringResult{
		SystemMonitorId: service.SystemMonitorId.String(),
		ServicePluginID: p.Name(),
		HealthReport:    constants.GetStatusInfo(constants.UnknownStatus, ""),
		LastCheckTime:   time.Now(),
	}

	start := time.Now()

	if service.Device == monitors.ServiceMonitorWebModules {
		client := &http.Client{
			Timeout: p.timeout,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return net.DialTimeout(network, addr, p.timeout)
				},
			},
		}

		httpUrl := p.formatHTTPAddress(service.Host, service.Port)
		_, err := url.Parse(httpUrl)
		if err != nil {
			status.FailureCount++
			status.HealthReport = constants.GetStatusInfo(constants.InvalidConfiguration, "")
			return status, fmt.Errorf("invalid URL format: %v", err)
		}

		// Get HTTP method from config (default to GET)
		method := "GET"
		if m, ok := p.config["method"].(string); ok {
			method = strings.ToUpper(m)
		}

		var req *http.Request
		var body io.Reader = nil

		// Handle POST requests if configured
		if method == "POST" {
			if postData, ok := p.config["post_data"].(string); ok {
				body = strings.NewReader(postData)
			}
		}

		// Create the HTTP request
		req, err = http.NewRequestWithContext(ctx, method, httpUrl, body)
		if err != nil {
			status.FailureCount++
			status.HealthReport = constants.GetStatusInfo(constants.UnknownStatus, "")
			return status, fmt.Errorf("failed to create HTTP request: %v", err)
		}

		// Add authentication if credentials are provided
		if auth, ok := p.config["auth_token"]; ok {
			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", auth))
		}

		// Set headers if configured
		if headers, ok := p.config["headers"].(map[string]any); ok {
			for k, v := range headers {
				if vs, ok := v.(string); ok {
					req.Header.Set(k, vs)
				}
			}
		}

		// Set Content-Type for POST if not specified
		if method == "POST" && req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := client.Do(req)
		if err != nil {
			status.FailureCount++
			status.HealthReport = constants.GetStatusInfo(constants.Degraded, "")
			return status, fmt.Errorf("HTTP request failed: %v", err)
		}
		defer resp.Body.Close()

		responseTime := time.Since(start)
		status.Details["response_time_ms"] = float64(responseTime.Milliseconds())
		status.Details["status_code"] = float64(resp.StatusCode)
		status.Details["status_code"] = resp.StatusCode
		status.Details["response_time"] = responseTime.String()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			status.FailureCount = 0
			status.HealthReport = constants.GetStatusInfo(constants.Healthy, "")

			return status, nil
		} else if resp.StatusCode >= 300 && resp.StatusCode < 500 {
			status.FailureCount++
			status.HealthReport = constants.GetStatusInfo(constants.Escalation, fmt.Sprintf("Service returned status %d", resp.StatusCode))
		} else {
			status.FailureCount++
			status.HealthReport = constants.GetStatusInfo(constants.Escalation, fmt.Sprintf("Service returned status %d", resp.StatusCode))
		}

		return status, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	status.FailureCount = 0
	status.HealthReport = constants.GetStatusInfo(constants.Healthy, "2")

	return status, nil
}

func (p *HTTPMonitorPlugin) Name() string {
	return "http_monitor"
}

func (p *HTTPMonitorPlugin) Description() string {
	return "HTTP service monitor"
}

func (p *HTTPMonitorPlugin) SupportedTypes() []monitors.ServiceType {
	return []monitors.ServiceType{monitors.ServiceMonitorWebModules, monitors.ServiceMonitorServer}
}

func (hc *HTTPMonitorPlugin) Cleanup() error {
	return nil
}

var Plugin monitors.ServiceMonitorPlugin = NewHTTPMonitorPlugin()

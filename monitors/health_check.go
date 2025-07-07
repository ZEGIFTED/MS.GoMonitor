package monitors

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net"
	"strconv"

	"strings"
	"time"

	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
)

// DefaultChecker implements Basic Monitoring Logic
type HealthCheck struct {
	config map[string]any
}

// Check implements the default health check logic
func (hc *HealthCheck) Check(ctx context.Context, db *sql.DB, service ServiceMonitorData) (MonitoringResult, error) {
	log.Println("Running Default Health Check")

	status := MonitoringResult{
		SystemMonitorId: service.SystemMonitorId.String(),
		ServicePluginID: "Health Check",
		HealthReport:    constants.GetStatusInfo(constants.UnknownStatus, ""),
		LastCheckTime:   time.Now(),
	}

	if service.Host == "" {
		slog.Info("DEBUG FailureCount", "FailureCount", status.FailureCount)
		status.FailureCount++
		status.HealthReport = constants.GetStatusInfo(constants.InvalidConfiguration, "")
		return status, fmt.Errorf("host cannot be empty")
	}

	timeout := 15 * time.Second
	if t, ok := hc.config["timeout"].(float64); ok {
		timeout = time.Duration(t) * time.Second
	}

	// First check basic TCP connectivity
	tcpAddr := net.JoinHostPort(service.Host, strconv.Itoa(service.Port))
	conn, err := net.DialTimeout("tcp", tcpAddr, timeout)
	if err != nil {
		status.FailureCount++
		status.HealthReport = constants.GetStatusInfo(constants.Escalation, "")
		return status, fmt.Errorf("TCP connection failed: %v", err)
	}
	conn.Close()

	// err = hc.telnetCheck(service.Host, service.Port, timeout)
	// if err != nil {
	// 	status.FailureCount++
	// 	status.HealthReport = constants.GetStatusInfo(constants.Escalation, "")
	// 	return status, fmt.Errorf("telnet check failed: %v", err)
	// }

	status.FailureCount = 0
	status.HealthReport = constants.GetStatusInfo(constants.Healthy, "")

	return status, nil
}

func (p *HealthCheck) telnetCheck(host string, port int, timeout time.Duration) error {
	// Reduce dial timeout to 75% of total to leave room for read
	dialTimeout := timeout * 75 / 100
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), dialTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Set remaining time for read
	remainingTimeout := timeout - dialTimeout
	if remainingTimeout <= 0 {
		return fmt.Errorf("insufficient time remaining for read")
	}
	conn.SetReadDeadline(time.Now().Add(remainingTimeout))

	// Service-specific handling
	switch port {
	case 22: // SSH
		_, err = conn.Write([]byte("\n"))
		if err != nil {
			return err
		}
		buf := make([]byte, 1024)
		_, err = conn.Read(buf)
		if err != nil {
			return err
		}
		if !strings.Contains(string(buf), "SSH") {
			return fmt.Errorf("not an SSH service")
		}

	case 1433: // SQL Server
		// SQL Server needs a TDS pre-login packet to respond
		// Basic TDS pre-login header (simplified)
		prelogin := []byte{
			0x12, 0x01, 0x00, 0x34, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x15, 0x00, 0x06, 0x01, 0x00, 0x20,
			0x00, 0x01, 0x02, 0x00, 0x21, 0x00, 0x01, 0x03,
			0x00, 0x22, 0x00, 0x04, 0x04, 0x00, 0x26, 0x00,
			0x01, 0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		}
		_, err = conn.Write(prelogin)
		if err != nil {
			return err
		}
		buf := make([]byte, 1024)
		_, err = conn.Read(buf)
		if err != nil {
			return err
		}
		// Check for TDS response (first byte should be 0x04)
		if len(buf) > 0 && buf[0] != 0x04 {
			return fmt.Errorf("not a SQL Server service")
		}

	default:
		// Generic check - just verify TCP connectivity
		return nil
	}

	return nil
}

// Name returns the name of the default checker
func (hc *HealthCheck) Name() string {
	return "Service Health Check"
}

// Description returns a description of the default checker
func (hc *HealthCheck) Description() string {
	return "Default Service Health Checker"
}

// SupportedTypes returns the service types this checker supports
func (hc *HealthCheck) SupportedTypes() []ServiceType {
	return []ServiceType{
		ServiceMonitorAgent,
		ServiceMonitorWebModules,
		ServiceMonitorSNMP,
		ServiceMonitorServer,
	}
}

// Init initializes the default checker (no-op in this case)
func (hc *HealthCheck) Initialize(config map[string]any) error {
	log.Println("Initializing HTTP/TCP/Telnet Monitor Plugin")

	return nil
}

func (hc *HealthCheck) Cleanup() error {
	return nil
}

package sslcheck

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net"
	"time"

	"github.com/ZEGIFTED/MS.GoMonitor/monitors"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
)

type SSLChecker struct {
	config map[string]any
}

func NewSSLChecker() *SSLChecker {
	return &SSLChecker{}
}

func (s *SSLChecker) Initialize(config map[string]any) error {
	s.config = config
	return nil
}

func (s *SSLChecker) Check(ctx context.Context, db *sql.DB, service monitors.ServiceMonitorData) (monitors.MonitoringResult, error) {
	status := monitors.MonitoringResult{
		SystemMonitorId: service.SystemMonitorId.String(),
		ServicePluginID: s.Name(),
		HealthReport:    constants.GetStatusInfo(constants.UnknownStatus, ""),
		LastCheckTime:   time.Now(),
	}

	timeout := 10 * time.Second
	if t, ok := s.config["timeout"].(float64); ok {
		timeout = time.Duration(t) * time.Second
	}

	host := service.Host
	if host == "" {
		status.FailureCount++
		status.HealthReport = constants.GetStatusInfo(constants.InvalidConfiguration, "")
		return status, fmt.Errorf("host cannot be empty")
	}

	port := service.Port
	if port == 0 {
		port = 443 // Default HTTPS port
	}

	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: timeout},
		"tcp",
		fmt.Sprintf("%s:%d", host, port),
		&tls.Config{InsecureSkipVerify: false},
	)
	if err != nil {
		status.FailureCount++
		status.HealthReport = constants.GetStatusInfo(constants.Escalation, "SSL connection Failed "+err.Error())
		return status, fmt.Errorf("SSL connection Failed. %s:%v ..... %s", host, port, err.Error())
	}
	defer conn.Close()

	// Verify certificate expiration
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		status.FailureCount++
		status.HealthReport = constants.GetStatusInfo(constants.Escalation, "No SSL Certificates Found")
		return status, fmt.Errorf("PLUGIN [%s] CHECK Failed for %s. Message -> No SSL Certificates Found", s.Name(), service.Name)
	}

	expiry := certs[0].NotAfter
	daysUntilExpiry := int(time.Until(expiry).Hours() / 24)

	if daysUntilExpiry <= 0 {
		status.FailureCount++
		status.HealthReport = constants.GetStatusInfo(constants.Escalation, fmt.Sprintf("certificate expired %d days ago", -daysUntilExpiry))
		return status, fmt.Errorf("certificate expired %d days ago", -daysUntilExpiry)
	}

	if daysUntilExpiry <= 7 { // Warn if expiring within 7 days
		status.FailureCount++
		status.HealthReport = constants.GetStatusInfo(constants.Degraded, fmt.Sprintf("certificate is expiring in %d days", -daysUntilExpiry))
		return status, fmt.Errorf("certificate expires in %d days", daysUntilExpiry)
	}

	status.FailureCount = 0
	status.HealthReport = constants.GetStatusInfo(constants.Healthy, "")

	return status, nil
}

func (s *SSLChecker) Name() string {
	return "ssl_check"
}

func (s *SSLChecker) Description() string {
	return "Checks SSL/TLS certificate validity and expiration"
}

func (s *SSLChecker) SupportedTypes() []monitors.ServiceType {
	return []monitors.ServiceType{
		monitors.ServiceMonitorWebModules,
		monitors.ServiceMonitorServer,
	}
}

func (s *SSLChecker) Cleanup() error {
	return nil
}

var Plugin monitors.ServiceMonitorPlugin = NewSSLChecker()

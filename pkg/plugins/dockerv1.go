package plugins

import (
	"context"
	"database/sql"
	"time"

	"github.com/ZEGIFTED/MS.GoMonitor/monitors"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
)

type DockerPlugin struct {
	config map[string]any
}

func (d *DockerPlugin) Initialize(config map[string]interface{}) error {
	d.config = config
	return nil
}

func _DockerPlugin() *DockerPlugin {
	return &DockerPlugin{}
}

func (p *DockerPlugin) Name() string {
	return "docker"
}

func (p *DockerPlugin) Description() string {
	return "Docker Engine"
}

func (p *DockerPlugin) SupportedTypes() []monitors.ServiceType {
	return []monitors.ServiceType{monitors.ServiceMonitorServer}
}

func (hc *DockerPlugin) Cleanup() error {
	return nil
}

func (d *DockerPlugin) Check(ctx context.Context, db *sql.DB, service monitors.ServiceMonitorData) (monitors.MonitoringResult, error) {
	status := monitors.MonitoringResult{
		SystemMonitorId: service.SystemMonitorId.String(),
		ServicePluginID: d.Name(),
		HealthReport:    constants.GetStatusInfo(constants.UnknownStatus, ""),
		LastCheckTime:   time.Now(),
	}

	status.FailureCount = 0
	status.HealthReport = constants.GetStatusInfo(constants.Healthy, "")

	return status, nil
}

var Docker monitors.ServiceMonitorPlugin = _DockerPlugin()

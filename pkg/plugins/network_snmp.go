package plugins

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net"
	"strings"

	// "os"
	"time"

	"github.com/ZEGIFTED/MS.GoMonitor/internal/repository"
	"github.com/ZEGIFTED/MS.GoMonitor/monitors"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
)

type NetworkSNMPPlugin struct {
	config map[string]any
}

func NewNetworkSNMPPlugin() *NetworkSNMPPlugin {
	return &NetworkSNMPPlugin{}
}

func (p *NetworkSNMPPlugin) Initialize(config map[string]interface{}) error {
	p.config = config
	return nil
}

func (s *NetworkSNMPPlugin) Check(ctx context.Context, db *sql.DB, netDevice monitors.ServiceMonitorData) (monitors.MonitoringResult, error) {
	status := monitors.MonitoringResult{
		SystemMonitorId: netDevice.SystemMonitorId.String(),
		ServicePluginID: s.Name(),
		HealthReport:    constants.GetStatusInfo(constants.UnknownStatus, ""),
		LastCheckTime:   time.Now(),
	}

	if netDevice.Host == "" && netDevice.Port == 0 {
		status.FailureCount++
		status.HealthReport = constants.GetStatusInfo(constants.InvalidConfiguration, "Invalid SNMP configuration")
		return status, fmt.Errorf("Host or Port cannot be empty")
	}

	var config monitors.SNMPDeviceConfig

	if communityString, ok := netDevice.Configuration["communityString"]; ok {
		config.CommunityString, _ = communityString.(string)
	}

	if authUsername, ok := netDevice.Configuration["authUsernameV3"]; ok {
		config.AuthUsernameV3, _ = authUsername.(string)
	}

	if ap, ok := netDevice.Configuration["authPasswordV3"]; ok {
		config.AuthPasswordV3, _ = ap.(string)
	}

	if pp, ok := netDevice.Configuration["privPassword"]; ok {
		config.PrivPassword, _ = pp.(string)
	}

	if sv, ok := netDevice.Configuration["snmpVersion"]; ok {
		config.SNMPVersion, _ = sv.(string)
		config.SNMPVersion = strings.ToLower(strings.TrimSpace(config.SNMPVersion))
	} else {
		config.SNMPVersion = "v2" // default
	}

	// Handle SNMP metrics configuration
	var metricConfiguration []monitors.SNMPMetricConfig

	snmpMetrics, ok := netDevice.Configuration["snmpMetrics"]
	if ok {
		// Try to unmarshal if it's a JSON string
		if metricsStr, ok := snmpMetrics.(string); ok {
			err := json.Unmarshal([]byte(metricsStr), &metricConfiguration)
			if err != nil {
				// Handle error or use default metrics
				log.Printf("Error parsing snmpMetrics: %v", err)
			}
		} else if metricsSlice, ok := snmpMetrics.([]any); ok {
			// Handle case where it's already a slice
			for _, m := range metricsSlice {
				if metricMap, ok := m.(map[string]any); ok {
					mc := monitors.SNMPMetricConfig{
						OID:         getString(metricMap, "oid"),
						Description: getString(metricMap, "description"),
						MetricType:  getString(metricMap, "metricType"),
						Name:        getString(metricMap, "name"),
					}
					// if scale, ok := metricMap["scale"].(float64); ok {
					// 	config.Scale = scale
					// }
					if unit, ok := metricMap["unit"].(string); ok {
						mc.Unit = unit
					}
					// metricConfiguration = append(metricConfiguration, mc)
					config.SNMPMetrics = append(config.SNMPMetrics, mc)
				}
			}
		}
	}

	// If no metrics configured, use some defaults
	if len(metricConfiguration) == 0 {
		slog.Info("Using Defalt Metric OID Configuration")
		metricConfiguration = []monitors.SNMPMetricConfig{
			{
				OID:         "1.3.6.1.2.1.1.5.0",
				Description: "System Name",
				MetricType:  "string",
				Name:        "sysName",
			},
			{
				OID:         "1.3.6.1.2.1.1.1.0",
				Description: "System Description",
				MetricType:  "string",
				Name:        "sysDescr",
			},
			// Add more default metrics as needed
		}
	}

	// Define metrics to collect
	// metrics = []struct {
	// 	OID         string
	// 	Description string
	// 	MetricType  string
	// }{
	// 	// System information
	// 	{".1.3.6.1.2.1.197.1.1.3.0", "manufacturer", "system"},
	// 	{".1.3.6.1.2.1.47.1.1.1.1.11.67109120", "serial_number", "system"},
	// 	{".1.3.6.1.2.1.47.1.1.1.1.10.67109120", "software_version", "system"},
	// 	{".1.3.6.1.2.1.1.1.0", "system_description", "system"},
	// 	{".1.3.6.1.2.1.1.3.0", "uptime", "system"},
	// 	{".1.3.6.1.2.1.1.5.0", "system_name", "system"},
	// 	{".1.3.6.1.4.1.9", "location", "system"},

	// 	// Interface information
	// 	{".1.3.6.1.2.1.2.1.0", "interface_count", "interface"},
	// 	// 1.3.6.1.2.1.2.2.1.17.1
	// 	// .1.3.6.1.2.1.2.2.1.2

	// 	// Memory metrics
	// 	{".1.3.6.1.2.1.25.2.2.0", "memory_total", "memory"},
	// 	{".1.3.6.1.2.1.25.2.3.1.6.1", "memory_used", "memory"},

	// 	// CPU metrics
	// 	{".1.3.6.1.2.1.25.2.3.1.5", "cpu_usage", "cpu"},
	// 	{".1.3.6.1.2.1.25.3.3.1.2", "cpu_load", "cpu"},

	// 	// Network traffic
	// 	{".1.3.6.1.2.1.31.1.1.1.6.1", "inbound_traffic", "traffic"},
	// 	{".1.3.6.1.2.1.31.1.1.1.10.1", "outbound_traffic", "traffic"},
	// 	// gig0_0_in_oct = '1.3.6.1.2.1.2.2.1.10.1'
	// 	// gig0_0_in_uPackets = '1.3.6.1.2.1.2.2.1.11.1'
	// 	// gig0_0_out_oct = '1.3.6.1.2.1.2.2.1.16.1'
	// 	// gig0_0_out_uPackets = '1.3.6.1.2.1.2.2.1.17.1'
	// }

	// if ok {

	// 	//Parse configuration JSON
	// 	// metricConfiguration =
	// 	if v, ok := snmpMetrics.(string); ok {
	// 		if err := json.Unmarshal([]byte(v), &metricConfiguration); err != nil {
	// 			log.Printf("Warning: could not parse configuration for Network Device %s: %v", netDevice.Name, err)
	// 		}
	// 	}
	// }

	switch config.SNMPVersion {
	case "v2", "v2c":
		// For v2, we only need community string
		if config.CommunityString == "" {
			log.Println("Using default community string for SNMP v2")
			config.CommunityString = "public"
		}

	case "v3":
		// For v3, we need auth username and password
		if config.AuthUsernameV3 == "" || config.AuthPasswordV3 == "" {
			status.FailureCount++
			status.HealthReport = constants.GetStatusInfo(constants.InvalidConfiguration, "SNMP v3 requires authUsernameV3 and authPasswordV3")
			return status, fmt.Errorf("SNMP v3 requires authUsernameV3 and authPasswordV3")
		}
	}

	networkManager := &monitors.NetworkManager{
		Target:       netDevice.Host,
		AuthUser:     config.AuthUsernameV3,
		AuthPassword: config.AuthPasswordV3,
		PrivPassword: config.PrivPassword,
		Community:    config.CommunityString,
	}

	snmpClient := networkManager.SNMPClient(config.SNMPVersion)

	snmpConnectErr := snmpClient.SNMP.Connect()
	if snmpConnectErr != nil {
		slog.ErrorContext(ctx, "SNMP Connection Error", snmpConnectErr.Error(), "Error")

		status.FailureCount++
		status.HealthReport = constants.GetStatusInfo(constants.Degraded, "Error Connecting to Network Device SNMP "+snmpConnectErr.Error())
		return status, snmpConnectErr
	}

	defer func(Conn net.Conn) {
		err := Conn.Close()
		if err != nil {
			fmt.Printf("Error closing connection: %v", err)
		}
	}(snmpClient.SNMP.Conn)

	// Collect metrics
	deviceMetrics, err := networkManager.CollectSNMPMetrics(snmpClient.SNMP, netDevice.SystemMonitorId.String(), netDevice.Host, config.SNMPMetrics)
	if err != nil {
		slog.ErrorContext(ctx, "Metric collection failed: ", "Error", err.Error())

		status.FailureCount++
		status.HealthReport = constants.GetStatusInfo(constants.Escalation, "Metric collection failed: "+err.Error())
		return status, err
	}

	// for _, metric := range config.SNMPMetrics {
	// 	result, err := snmpClient.snmp.Get([]string{metric.OID})
	// 	if err != nil {
	// 		log.Printf("Error getting %s: %v\n", metric.Description, err)
	// 		lastError = err
	// 		continue

	// 		// return monitors.ServiceMonitorStatus{
	// 		// 	Name:          netDevice.Name,
	// 		// 	Device:        netDevice.Device,
	// 		// 	LiveCheckFlag: constants.Degraded,
	// 		// 	Status:        "Error retrieving metrics",
	// 		// 	LastCheckTime: time.Now(),
	// 		// 	FailureCount:  1,
	// 		// }, false
	// 	}

	// 	name, value := GetSNMPValue(result.Variables)
	// 	if name == "" {
	// 		continue
	// 	}

	// 	log.Println(result.AgentAddress, metric.Description, name, value)
	// 	deviceMetrics = append(deviceMetrics, mstypes.NetworkDeviceMetric{
	// 		SystemMonitorId: netDevice.SystemMonitorId.String(),
	// 		DeviceIP:        netDevice.Host,
	// 		// DeviceType:      string(config.Device),
	// 		// MetricType:      metric.MetricType,
	// 		MetricName:        name,
	// 		MetricDescription: metric.Description,
	// 		MetricValue:       value,
	// 		LastPoll:          time.Now().Format(time.DateTime),
	// 	})

	// 	log.Printf("Collected metric: %s = %v\n", metric.Description, value)
	// }

	// GetInterfaces(snmpClient.snmp)

	// Sync metrics to database
	if err := repository.SyncNetworkMetrics(db, deviceMetrics); err != nil {
		slog.ErrorContext(ctx, "Error syncing metrics to database:", "Error", err.Error())

		status.FailureCount++
		status.HealthReport = constants.GetStatusInfo(constants.Escalation, "Error saving metrics to database "+err.Error())
		return status, err
	}

	status.FailureCount = 0
	status.HealthReport = constants.GetStatusInfo(constants.Healthy, "")

	return status, nil
}

// Helper function to format timeticks into a human-readable duration

// Helper function to safely get strings from map
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func (p *NetworkSNMPPlugin) Name() string {
	return "network_snmp"
}

func (p *NetworkSNMPPlugin) Description() string {
	return "SNMP Network Data Collector"
}

func (p *NetworkSNMPPlugin) SupportedTypes() []monitors.ServiceType {
	return []monitors.ServiceType{monitors.ServiceMonitorWebModules, monitors.ServiceMonitorServer}
}

func (hc *NetworkSNMPPlugin) Cleanup() error {
	return nil
}

var NetworkSNMP monitors.ServiceMonitorPlugin = NewNetworkSNMPPlugin()

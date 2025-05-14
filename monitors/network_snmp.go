package monitors

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net"
	"strconv"
	"strings"

	// "os"
	"time"

	"github.com/ZEGIFTED/MS.GoMonitor/internal/repository"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
	mstypes "github.com/ZEGIFTED/MS.GoMonitor/types"
	"github.com/gosnmp/gosnmp"
)

// Define a struct for SNMP metric configuration
type SNMPMetricConfig struct {
	OID         string `json:"oid"`
	Description string `json:"description"`
	MetricType  string `json:"metricType"`
	// Add additional fields as needed

	IsCounter   bool   `json:"isCounter"`   // For metrics that need delta calculation
	Scale       int    `json:"scale"`       // Scaling factor (e.g., 1000 for KB, 1000000 for MB)
	InterfaceID string `json:"interfaceId"` // For interface-specific metrics

	Name string `json:"name"`
	// Scale       float64 `json:"scale,omitempty"`
	Unit string `json:"unit,omitempty"`
}

type SNMPDeviceConfig struct {
	CommunityString string             `json:"communityString"`
	AuthUsernameV3  string             `json:"authUsernameV3"`
	AuthPasswordV3  string             `json:"authPasswordV3"`
	PrivPassword    string             `json:"privPassword"`
	SNMPVersion     string             `json:"snmpVersion"`
	SNMPMetrics     []SNMPMetricConfig `json:"snmpMetrics"`
}

func (service *SNMPServiceChecker) Check(netDevice ServiceMonitorData, ctx context.Context, db *sql.DB) (ServiceMonitorStatus, bool) {
	if netDevice.Host == "" && netDevice.Port == 0 {
		return ServiceMonitorStatus{
			Name:          netDevice.Name,
			Device:        netDevice.Device,
			LiveCheckFlag: constants.Degraded,
			Status:        "Invalid SNMP configuration",
			LastCheckTime: time.Now(),
			FailureCount:  1,
		}, false
	}

	var config SNMPDeviceConfig

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
	var metricConfiguration []SNMPMetricConfig

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
					mc := SNMPMetricConfig{
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
		metricConfiguration = []SNMPMetricConfig{
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

		// // Clear any v3 credentials if they were provided
		// authUser = ""
		// authPass = ""
		// privPass = ""

	case "v3":
		// For v3, we need auth username and password
		if config.AuthUsernameV3 == "" || config.AuthPasswordV3 == "" {
			return ServiceMonitorStatus{
				Name:          netDevice.Name,
				Device:        netDevice.Device,
				LiveCheckFlag: constants.Escalation,
				Status:        "SNMP v3 requires authUsernameV3 and authPasswordV3",
				LastCheckTime: time.Now(),
				FailureCount:  1,
			}, false
		}
	}

	snmpHandler := &NetworkManager{
		Target:       netDevice.Host,
		AuthUser:     config.AuthUsernameV3,
		AuthPassword: config.AuthPasswordV3,
		PrivPassword: config.PrivPassword,
		Community:    config.CommunityString,
	}

	snmpClient := snmpHandler.SNMPClient(config.SNMPVersion)

	snmpConnectErr := snmpClient.snmp.Connect()
	if snmpConnectErr != nil {
		slog.InfoContext(ctx, "SNMP Connection Error", snmpConnectErr.Error(), "Error")
		return ServiceMonitorStatus{
			Name:          netDevice.Name,
			Device:        netDevice.Device,
			LiveCheckFlag: constants.Escalation,
			Status:        "Error Connecting to Network Device SNMP " + snmpConnectErr.Error(),
			LastCheckTime: time.Now(),
			FailureCount:  1,
		}, false
	}

	defer func(Conn net.Conn) {
		err := Conn.Close()
		if err != nil {
			fmt.Printf("Error closing connection: %v", err)
		}
	}(snmpClient.snmp.Conn)

	// Collect metrics
	deviceMetrics, err := collectSNMPMetrics(snmpClient.snmp, netDevice.SystemMonitorId.String(), netDevice.Host, config.SNMPMetrics)
	if err != nil {
		log.Println("Metric collection failed: ", err)
		return ServiceMonitorStatus{
			Name:          netDevice.Name,
			Device:        netDevice.Device,
			LiveCheckFlag: constants.Degraded,
			Status:        "Metric collection failed: " + err.Error(),
			LastCheckTime: time.Now(),
			FailureCount:  1,
		}, false
	}

	// for _, metric := range config.SNMPMetrics {
	// 	result, err := snmpClient.snmp.Get([]string{metric.OID})
	// 	if err != nil {
	// 		log.Printf("Error getting %s: %v\n", metric.Description, err)
	// 		lastError = err
	// 		continue

	// 		// return ServiceMonitorStatus{
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
		log.Println("Error syncing metrics to database:", err)
		return ServiceMonitorStatus{
			Name:          netDevice.Name,
			Device:        netDevice.Device,
			LiveCheckFlag: constants.Degraded,
			Status:        "Error saving metrics to database " + err.Error(),
			LastCheckTime: time.Now(),
			FailureCount:  1,
		}, false
	}

	return ServiceMonitorStatus{
		Name:              netDevice.Name,
		Device:            netDevice.Device,
		LiveCheckFlag:     constants.Healthy,
		Status:            "Successfully collected " + strconv.Itoa(len(deviceMetrics)) + " metrics",
		LastCheckTime:     time.Now(),
		LastServiceUpTime: time.Now(),
		FailureCount:      0,
	}, true
}

func collectSNMPMetrics(snmp *gosnmp.GoSNMP, SystemMonitorId, Host string, metrics []SNMPMetricConfig) ([]mstypes.NetworkDeviceMetric, error) {
	var deviceMetrics []mstypes.NetworkDeviceMetric
	var oids []string
	oidToMetric := make(map[string]SNMPMetricConfig)

	// Prepare OIDs to query
	for _, metric := range metrics {
		oids = append(oids, metric.OID)
		oidToMetric[metric.OID] = metric
	}

	// Perform SNMP Get request
	packet, err := snmp.Get(oids)
	if err != nil {
		return nil, fmt.Errorf("SNMP get failed: %v", err)
	}

	// Process results
	for _, variable := range packet.Variables {
		if metric, exists := oidToMetric[variable.Name]; exists {
			value, err := convertSNMPValue(variable)
			if err != nil {
				log.Printf("Failed to convert value for OID %s: %v", variable.Name, err)
				continue
			}

			// Apply scaling if configured
			if metric.Scale > 0 {
				if num, ok := value.(int); ok {
					value = num / metric.Scale
				} else if num, ok := value.(float64); ok {
					value = num / float64(metric.Scale)
				}
			}

			slog.Info("Device Metrics", SystemMonitorId, deviceMetrics)
			deviceMetrics = append(deviceMetrics, mstypes.NetworkDeviceMetric{
				SystemMonitorId: SystemMonitorId,
				DeviceIP:        Host,
				// DeviceType:      string(config.Device),
				// MetricType:      metric.MetricType,
				MetricName:        metric.Name,
				MetricDescription: metric.Description,
				MetricValue:       value,
				LastPoll:          time.Now().Format(time.DateTime),
			})

			// results = append(results, mstypes.NetworkDeviceMetric{
			//     OID:         variable.Name,
			//     Description: metric.Description,
			//     Value:       value,
			//     MetricType:  metric.MetricType,
			//     Timestamp:  time.Now(),
			// })
		}
	}

	return deviceMetrics, nil
}

func convertSNMPValue(variable gosnmp.SnmpPDU) (interface{}, error) {
	switch variable.Type {
	case gosnmp.Integer, gosnmp.Counter32, gosnmp.Gauge32, gosnmp.TimeTicks, gosnmp.Counter64:
		return variable.Value, nil
	case gosnmp.OctetString:
		return string(variable.Value.([]byte)), nil
	case gosnmp.IPAddress:
		return net.IP(variable.Value.([]byte)).String(), nil
	case gosnmp.ObjectIdentifier:
		return variable.Value.(string), nil
	default:
		return nil, fmt.Errorf("unsupported SNMP type %v", variable.Type)
	}
}

func GetSNMPValue(result []gosnmp.SnmpPDU) (string, string) {
	for _, variable := range result {
		log.Printf("OID: %s\n", variable.Name)
		switch variable.Type {
		case gosnmp.OctetString:
			return variable.Name, string(variable.Value.([]byte))
		case gosnmp.TimeTicks:
			if value, ok := variable.Value.(uint32); ok {
				return variable.Name, formatTimeticks(value)
				// return formatTimeticks(gosnmp.ToBigInt(value))
			}
		case gosnmp.Integer:
			if value, ok := variable.Value.(int); ok {
				return variable.Name, strconv.Itoa(value)
			}
		case gosnmp.Counter32, gosnmp.Gauge32:
			if value, ok := variable.Value.(uint32); ok {
				return variable.Name, strconv.FormatUint(uint64(value), 10)
			}
		case gosnmp.Counter64:
			if value, ok := variable.Value.(uint64); ok {
				return variable.Name, strconv.FormatUint(value, 10)
			}
		case gosnmp.Null:
			return variable.Name, variable.Value.(string)
		default:
			// Fallback for unhandled types or conversion failures
			return variable.Name, fmt.Sprintf("%v", variable.Value)
		}
	}

	if len(result) == 0 {
		return "", ""
	}

	variable := result[0]

	// Fallback for unhandled types or conversion failures
	return variable.Name, fmt.Sprintf("%v", variable.Value)
}

// Helper function to format timeticks into a human-readable duration
func formatTimeticks(ticks uint32) string {
	seconds := float64(ticks) / 100
	duration := time.Duration(seconds * float64(time.Second))
	return duration.Abs().String()
}

// Helper function to safely get strings from map
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func GetInterfaces(snmp *gosnmp.GoSNMP) ([]string, error) {
	var interfaces []string

	//err_ := snmp.Walk("1.3.6.1.2.1.2.2.1", func(variable gosnmp.SnmpPDU) error {
	//	fmt.Printf("OID: %s, Value: %v\n", variable.Name, variable.Value)
	//	return nil
	//})
	//if err_ != nil {
	//	log.Fatalf("Error walking SNMP table: %v", err_)
	//}

	// Get interface descriptions
	// Define the walk function
	walkFn := func(pdu gosnmp.SnmpPDU) error {
		// Create a single-element slice containing the PDU to pass to GetSNMPValue
		result := []gosnmp.SnmpPDU{pdu}

		// Get the human-readable value
		name, value := GetSNMPValue(result)

		// Log the OID, value, and type
		log.Printf("OID: %s, Value: %s -> %s, Type: %v", pdu.Name, name, value, pdu.Type)

		return nil
	}

	// Get interface descriptions using BulkWalk with callback
	err := snmp.BulkWalk(".1.3.6.1.2.1.25.2", walkFn)
	if err != nil {
		return nil, fmt.Errorf("BulkWalk error: %v", err)
	}

	return interfaces, nil
}

package monitors

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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

func (service *SNMPServiceChecker) Check(netDevice ServiceMonitorData, _ context.Context, db *sql.DB) (ServiceMonitorStatus, bool) {
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

	communityString, ok := netDevice.Configuration["communityString"]
	community := ""
	if ok {
		community, _ = communityString.(string)
	}

	authUsername, ok := netDevice.Configuration["authUsernameV3"]
	authUser := ""
	if ok {
		authUser, _ = authUsername.(string)
	}

	authPassword, ok := netDevice.Configuration["authPasswordV3"]
	authPass := ""
	if ok {
		authPass, _ = authPassword.(string)
	}

	privPassword, ok := netDevice.Configuration["privPassword"]
	privPass := ""
	if ok {
		privPass, _ = privPassword.(string)
	}

	snmpVersion, ok := netDevice.Configuration["snmpVersion"]
	version := "v2" // default version
	if ok {
		if v, ok := snmpVersion.(string); ok {
			version = strings.ToLower(strings.TrimSpace(v))
		}
	}

	switch version {
	case "v2", "v2c":
		// For v2, we only need community string
		if community == "" {
			log.Println("Using default community string for SNMP v2")
			community = "public"
		}

		// Clear any v3 credentials if they were provided
		authUser = ""
		authPass = ""
		privPass = ""

	case "v3":
		// For v3, we need auth username and password
		if authUser == "" || authPass == "" {
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
		AuthUser:     authUser,
		AuthPassword: authPass,
		PrivPassword: privPass,
		Community:    community,
	}

	snmpClient := snmpHandler.SNMPClient(version)

	snmpConnectErr := snmpClient.snmp.Connect()
	if snmpConnectErr != nil {
		log.Println("SNMP Connection Error", snmpConnectErr.Error())
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

	// Define metrics to collect
	metrics := []struct {
		OID         string
		Description string
		MetricType  string
	}{
		// System information
		{".1.3.6.1.2.1.197.1.1.3.0", "manufacturer", "system"},
		{".1.3.6.1.2.1.47.1.1.1.1.11.67109120", "serial_number", "system"},
		{".1.3.6.1.2.1.47.1.1.1.1.10.67109120", "software_version", "system"},
		{".1.3.6.1.2.1.1.1.0", "system_description", "system"},
		{".1.3.6.1.2.1.1.3.0", "uptime", "system"},
		{".1.3.6.1.2.1.1.5.0", "system_name", "system"},
		{".1.3.6.1.4.1.9", "location", "system"},

		// Interface information
		{".1.3.6.1.2.1.2.1.0", "interface_count", "interface"},
		// 1.3.6.1.2.1.2.2.1.17.1
		// .1.3.6.1.2.1.2.2.1.2

		// Memory metrics
		{".1.3.6.1.2.1.25.2.2.0", "memory_total", "memory"},
		{".1.3.6.1.2.1.25.2.3.1.6.1", "memory_used", "memory"},

		// CPU metrics
		{".1.3.6.1.2.1.25.2.3.1.5", "cpu_usage", "cpu"},
		{".1.3.6.1.2.1.25.3.3.1.2", "cpu_load", "cpu"},

		// Network traffic
		{".1.3.6.1.2.1.31.1.1.1.6.1", "inbound_traffic", "traffic"},
		{".1.3.6.1.2.1.31.1.1.1.10.1", "outbound_traffic", "traffic"},
		// gig0_0_in_oct = '1.3.6.1.2.1.2.2.1.10.1'
		// gig0_0_in_uPackets = '1.3.6.1.2.1.2.2.1.11.1'
		// gig0_0_out_oct = '1.3.6.1.2.1.2.2.1.16.1'
		// gig0_0_out_uPackets = '1.3.6.1.2.1.2.2.1.17.1'
	}

	// Collect metrics
	var deviceMetrics []mstypes.NetworkDeviceMetric
	var lastError error

	for _, metric := range metrics {
		result, err := snmpClient.snmp.Get([]string{metric.OID})
		if err != nil {
			log.Printf("Error getting %s: %v\n", metric.Description, err)
			lastError = err
			continue

			// return ServiceMonitorStatus{
			// 	Name:          netDevice.Name,
			// 	Device:        netDevice.Device,
			// 	LiveCheckFlag: constants.Degraded,
			// 	Status:        "Error retrieving metrics",
			// 	LastCheckTime: time.Now(),
			// 	FailureCount:  1,
			// }, false
		}

		name, value := GetSNMPValue(result.Variables)
		if name == "" {
			continue
		}

		log.Println(result.AgentAddress, metric.Description, name, value)
		deviceMetrics = append(deviceMetrics, mstypes.NetworkDeviceMetric{
			SystemMonitorId: netDevice.SystemMonitorId.String(),
			DeviceIP:        netDevice.Host,
			// DeviceType:      string(config.Device),
			// MetricType:      metric.MetricType,
			MetricName:        name,
			MetricDescription: metric.Description,
			MetricValue:       value,
			LastPoll:          time.Now().Format(time.DateTime),
		})

		log.Printf("Collected metric: %s = %v\n", metric.Description, value)
	}

	GetInterfaces(snmpClient.snmp)

	// Sync metrics to database
	if err := repository.SyncNetworkMetrics(db, deviceMetrics); err != nil {
		log.Println("Error syncing metrics to database:", err)
		return ServiceMonitorStatus{
			Name:          netDevice.Name,
			Device:        netDevice.Device,
			LiveCheckFlag: constants.Degraded,
			Status:        "Error saving metrics to database",
			LastCheckTime: time.Now(),
			FailureCount:  1,
		}, false
	}

	// Determine status based on errors
	if lastError != nil {
		return ServiceMonitorStatus{
			Name:          netDevice.Name,
			Device:        netDevice.Device,
			LiveCheckFlag: constants.Degraded,
			Status:        "Partial success - some metrics failed",
			LastCheckTime: time.Now(),
			FailureCount:  0, // Not a complete failure
		}, true
	}

	return ServiceMonitorStatus{
		Name:              netDevice.Name,
		Device:            netDevice.Device,
		LiveCheckFlag:     constants.Healthy,
		Status:            "Healthy",
		LastCheckTime:     time.Now(),
		LastServiceUpTime: time.Now(),
		FailureCount:      0,
	}, true
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

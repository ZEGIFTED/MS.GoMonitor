package monitors

import (
	"fmt"
	"log"
	"log/slog"
	"net"

	"strconv"
	"strings"
	"time"

	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
	mstypes "github.com/ZEGIFTED/MS.GoMonitor/types"
	"github.com/gosnmp/gosnmp"
)

type NetworkManager struct {
	SNMP         *gosnmp.GoSNMP
	trap         *gosnmp.TrapListener
	Target       string
	Community    string
	TrapPort     int
	AuthUser     string
	AuthPassword string
	PrivPassword string
}

// Define a struct for SNMP metric configuration
type SNMPMetricConfig struct {
	OID         string `json:"oid"`
	Description string `json:"description"`
	MetricType  string `json:"metricType"`

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

func (nm *NetworkManager) SNMPClient(deviceVersion string) *NetworkManager {
	var SNMP gosnmp.GoSNMP

	if strings.Contains(deviceVersion, "v2") {
		log.Println("Using SNMP v2")

		SNMP = gosnmp.GoSNMP{
			Target:    nm.Target, // Replace with your device's IP
			Port:      161,
			Community: nm.Community,
			Version:   gosnmp.Version2c,
			Timeout:   time.Duration(15) * time.Second,
			Retries:   constants.MaxRetries,
			// Logger:  gosnmp.NewLogger(log.New(os.Stdout, "", 0)),
		}
	} else {
		SNMP = gosnmp.GoSNMP{
			Target:        nm.Target,
			Port:          uint16(nm.TrapPort),
			Version:       gosnmp.Version3,
			Timeout:       time.Duration(35) * time.Second,
			SecurityModel: gosnmp.Default.SecurityModel,
			MsgFlags:      gosnmp.AuthNoPriv, // Default to AuthNoPriv. Use AuthPriv for both authentication and privacy
			SecurityParameters: &gosnmp.UsmSecurityParameters{
				UserName:                 nm.AuthUser,
				AuthenticationProtocol:   gosnmp.SHA512, // Use SHA for authentication
				AuthenticationPassphrase: nm.AuthPassword,
				PrivacyProtocol:          gosnmp.AES256, // Use AES for privacy
				PrivacyPassphrase:        nm.PrivPassword,
				AuthoritativeEngineTime:  uint32(time.Now().Unix()),
			},
			Retries: constants.MaxRetries,
			// Logger:  gosnmp.NewLogger(log.New(os.Stdout, "", 0)),
		}

		if nm.PrivPassword != "" {
			SNMP.MsgFlags = gosnmp.AuthPriv
		}
	}

	return &NetworkManager{
		SNMP: &SNMP,
	}
}

func (nm *NetworkManager) CollectSNMPMetrics(snmp *gosnmp.GoSNMP, SystemMonitorId, Host string, metrics []SNMPMetricConfig) ([]mstypes.NetworkDeviceMetric, error) {
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
			value, err := nm.ConvertSNMPValue(variable)
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

func (nm *NetworkManager) GetSNMPValue(result []gosnmp.SnmpPDU) (string, string) {
	for _, variable := range result {
		log.Printf("OID: %s\n", variable.Name)
		switch variable.Type {
		case gosnmp.OctetString:
			return variable.Name, string(variable.Value.([]byte))
		case gosnmp.TimeTicks:
			if value, ok := variable.Value.(uint32); ok {
				return variable.Name, nm.formatTimeticks(value)
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

func (nm *NetworkManager) ConvertSNMPValue(variable gosnmp.SnmpPDU) (interface{}, error) {
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

func (nm *NetworkManager) GetInterfaces(snmp *gosnmp.GoSNMP) ([]string, error) {
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
		name, value := nm.GetSNMPValue(result)

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

func (nm *NetworkManager) GetFormattedSNMPValue(result []gosnmp.SnmpPDU) (string, string) {
	for _, variable := range result {
		switch variable.Type {
		case gosnmp.OctetString:
			return variable.Name, string(variable.Value.([]byte))
		case gosnmp.TimeTicks:
			if value, ok := variable.Value.(uint32); ok {
				return variable.Name, nm.formatTimeticks(value)
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
			return variable.Name, "null"
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

func (nm *NetworkManager) formatTimeticks(ticks uint32) string {
	seconds := float64(ticks) / 100
	duration := time.Duration(seconds * float64(time.Second))
	return duration.Abs().String()
}

func (nm *NetworkManager) NewTrapClient() *gosnmp.TrapListener {
	// Configure the SNMP trap listener

	return &gosnmp.TrapListener{
		OnNewTrap: nm.HandleTrap,
		Params: &gosnmp.GoSNMP{
			Port:               162, // Standard SNMP trap port
			Transport:          "udp",
			Version:            gosnmp.Version2c,
			Timeout:            time.Duration(30) * time.Second,
			Retries:            3,
			ExponentialTimeout: true,
			MaxOids:            gosnmp.MaxOids,
			// Logger:               gosnmp.NewLogger(log.New(os.Stdout, "", 0)),
			SecurityModel:      gosnmp.UserSecurityModel,
			MsgFlags:           gosnmp.AuthPriv,
			SecurityParameters: nil, // Set if using SNMPv3
			ContextEngineID:    "",
			ContextName:        "",
			// IsAuthentic:          false,
		},
	}
}

// StartListener begins listening for SNMP traps
func (nm *NetworkManager) StartListener() error {
	// Configure GoSNMP parameters
	gosnmp.Default.Port = uint16(nm.TrapPort)
	gosnmp.Default.Community = nm.Community
	gosnmp.Default.Version = gosnmp.Version2c

	// Create a trap listener
	trapListener := nm.NewTrapClient()
	nm.trap = trapListener

	// Start listening for traps
	log.Printf("Starting SNMP trap listener on port %d", nm.TrapPort)

	err := trapListener.Listen("0.0.0.0:162")

	if err != nil {
		return fmt.Errorf("trap listener error: %v", err)
	}

	return nil
}

func (nm *NetworkManager) StopListener() {
	log.Println("Stopping Trap Listener...")
	nm.trap.Close()
}

// handleTrap processes incoming SNMP traps
func (nm *NetworkManager) HandleTrap(packet *gosnmp.SnmpPacket, addr *net.UDPAddr) {
	log.Printf("Received trap from %s:\n", addr.IP)

	// Print basic trap information
	log.Printf("SNMP Version: %d\n", packet.Version)
	log.Printf("Community: %s\n", packet.Community)
	log.Printf("PDU Type: %s\n", packet.PDUType)

	// Process each variable binding in the trap
	name, value := nm.GetFormattedSNMPValue(packet.Variables)

	// Log the OID, value, and type
	log.Printf("OID: %s, Value: %s", name, value)

	log.Println("End of trap")
}

// Example OID mappings for common traps
// var trapOIDs = map[string]string{
// 	"1.3.6.1.6.3.1.1.5.1":   "coldStart",
// 	"1.3.6.1.6.3.1.1.5.2":   "warmStart",
// 	"1.3.6.1.6.3.1.1.5.3":   "linkDown",
// 	"1.3.6.1.6.3.1.1.5.4":   "linkUp",
// 	"1.3.6.1.6.3.1.1.5.5":   "authenticationFailure",
// }

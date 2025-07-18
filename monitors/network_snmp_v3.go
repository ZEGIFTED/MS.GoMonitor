package monitors

// import (
// 	"context"
// 	"database/sql"
// 	"fmt"
// 	"log"
// 	"net"
// 	"os"
// 	"time"

// 	"github.com/ZEGIFTED/MS.GoMonitor/internal/repository"
// 	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
// 	mstypes "github.com/ZEGIFTED/MS.GoMonitor/types"
// 	"github.com/gosnmp/gosnmp"
// )

// func (service *SNMPServiceChecker) Check(config ServiceMonitorData, _ context.Context, db *sql.DB) (ServiceMonitorStatus, bool) {

// 	// Define the SNMPv3 connection parameters
// 	target := config.Host
// 	port := 161
// 	username := "msmonitoring"
// 	authPassword := "CaMoniLeb" // Replace with your SNMPv3 authentication password
// 	privPassword := "CaMoniLeb" // Replace with your SNMPv3 privacy password

// 	if target == "" && port == 0 {
// 		return ServiceMonitorStatus{
// 			Name:          config.Name,
// 			Device:        config.Device,
// 			LiveCheckFlag: constants.Degraded,
// 			Status:        "Invalid SNMP configuration",
// 			LastCheckTime: time.Now(),
// 			FailureCount:  1,
// 		}, false
// 	}

// 	// SNMPMetric represents an SNMP OID and its description
// 	type SNMPMetric struct {
// 		OID         string
// 		Description string
// 	}

// 	// Common SNMP OIDs for system information
// 	var commonMetrics = []SNMPMetric{
// 		{".1.3.6.1.2.1.197.1.1.3.0", "Manufacturer"},
// 		{".1.3.6.1.2.1.47.1.1.1.1.11.67109120", "SN"},
// 		{".1.3.6.1.2.1.47.1.1.1.1.10.67109120", "SV"},
// 		{".1.3.6.1.2.1.1.1.0", "System Description"},
// 		{".1.3.6.1.2.1.1.3.0", "Uptime"},
// 		{".1.3.6.1.2.1.1.5.0", "System Name"},
// 		{".1.3.6.1.4.1.9", "Location"},
// 		{".1.3.6.1.2.1.2.1.0", "Number of Interfaces"},
// 		{".1.3.6.1.2.1.25.2.2.0", "Memory"},
// 		{".1.3.6.1.2.1.25.2.3.1.6.1", "Memory"},

// 		{".1.3.6.1.2.1.25.2.3.1.5", "cpu"},
// 		{".1.3.6.1.2.1.25.3.3.1.2", "cpu"},
// 		{".1.3.6.1.2.1.31.1.1.1.6.1", "Inbound traffic"},
// 		{".1.3.6.1.2.1.31.1.1.1.10.1", "Outbound traffic"},

// 		{".1.3.6.1.2.1.4.20.1.1.0", "ipAdEntLastRoute"},
// 		{".1.3.6.1.2.1.2.2.1.1.0", "IfDescr"},
// 		{".1.3.6.1.2.1.2.2.1.5.0", "IfSpeed"},

// 		{".1.3.6.1.2.1.2.2.1.10.0", "IfInOctets"},  // (inbound traffic)
// 		{".1.3.6.1.2.1.2.2.1.16.0", "IfOutOctets"}, // (outbound traffic)
// 		{".1.3.6.1.2.1.2.2.1.1.0", "IPTable"},
// 		{".1.3.6.1.2.1.4.20.1.1.0", "ipDefaultRoute"},

// 		{".1.3.6.1.2.1.2.2.1.3", "IfType"},
// 		{".1.3.6.1.2.1.2.2.1.4", "IfMtu"},
// 		{".1.3.6.1.2.1.2.2.1.14", "IfInErrors"},
// 		{".1.3.6.1.2.1.2.2.1.20", "IfOutErrors"},
// 		{".1.3.6.1.2.1.31.1.1.1.6", "IfHCInOctets"},   // (64-bit counters)
// 		{".1.3.6.1.2.1.31.1.1.1.10", "IfHCOutOctets"}, // (64-bit counters)
// 	}

// 	// const OIDs interface {} = (
// 	// 	".1.3.6.1.2.1.1.1.0",
// 	// 	".1.3.6.1.2.1.1.3.0" // 4118776 UPTIME
// 	// 	".1.3.6.1.2.1.1.5.0", // System Name
// 	// 	".1.3.6.1.2.1.1.6.0", // location

// 	// 	".1.3.6.1.2.1.25.2.3.1.4.1", // OID for physical memory utilization
// 	// 	".1.3.6.1.2.1.25.2.3.1.4.2", // OID for physical memory utilization
// 	// 	".1.3.6.1.2.1.25.2.3.1.4.3", // OID for physical memory utilization
// 	// 	".1.3.6.1.2.1.25.2.3.1.4.4", // OID for physical memory utilization
// 	// 	".1.3.6.1.2.1.25.2.3.1.4.5", // OID for physical memory utilization
// 	// 	// ".1.3.6.1.2.1.25.2.3.1.4.6", // OID for physical memory utilization

// 	// 	".1.3.6.1.2.1.25.2.3.1.6.1", // Memory Usage
// 	// 	".1.3.6.1.2.1.25.2.3.1.6.3", // Memory Usage

// 	// 	// ".1.3.6.1.2.1.25.2.3.1.2.5",
// 	// 	".1.3.6.1.2.1.25.2.3.1.3.1",
// 	// 	".1.3.6.1.2.1.25.2.3.1.3.2",
// 	// 	".1.3.6.1.2.1.25.2.3.1.3.3",
// 	// 	".1.3.6.1.2.1.25.2.3.1.3.4",
// 	// 	".1.3.6.1.2.1.25.2.3.1.3.5",
// 	// 	// ".1.3.6.1.2.1.25.2.3.1.3.6",
// 	// 	// ".1.3.6.1.2.1.25.2.3.1.3.7",
// 	// 	// ".1.3.6.1.2.1.25.2.3.1.3.8",

// 	// 	".1.3.6.1.2.1.25.2.3.1.5.1",
// 	// 	".1.3.6.1.2.1.25.2.3.1.5.2",
// 	// 	".1.3.6.1.2.1.25.2.3.1.5.3",
// 	// 	".1.3.6.1.2.1.25.2.3.1.5.4",
// 	// 	".1.3.6.1.2.1.25.2.3.1.5.5",
// 	// 	// ".1.3.6.1.2.1.25.2.3.1.5.6",
// 	// 	// ".1.3.6.1.2.1.25.2.3.1.5.7",

// 	// 	// ".1.3.6.1.2.1.25.2.3.1.8.1",
// 	// 	".1.3.6.1.2.1.25.2.3.1.7.2",
// 	// 	".1.3.6.1.2.1.25.2.3.1.7.5",
// 	// );

// 	//ifInOctetsOID  = "1.3.6.1.2.1.2.2.1.10.1" // Incoming traffic (Replace 1 with your interface index)
// 	//ifOutOctetsOID = "1.3.6.1.2.1.2.2.1.16.1" // Outgoing traffic

// 	//".1.3.6.1.2.1.2.2.1.10", // ifInOctets
// 	//	".1.3.6.1.2.1.2.2.1.16", // ifOutOctets
// 	//	".1.3.6.1.2.1.2.2.1.5",  // ifSpeed

// 	// Configure SNMP connection
// 	snmp := &gosnmp.GoSNMP{
// 		Target:        target,
// 		Port:          uint16(port),
// 		Version:       gosnmp.Version3,
// 		Timeout:       time.Duration(35) * time.Second,
// 		SecurityModel: gosnmp.Default.SecurityModel,
// 		MsgFlags:      gosnmp.Default.MsgFlags, // Use AuthPriv for both authentication and privacy
// 		SecurityParameters: &gosnmp.UsmSecurityParameters{
// 			UserName:                 username,
// 			AuthenticationProtocol:   gosnmp.SHA512, // Use SHA for authentication
// 			AuthenticationPassphrase: authPassword,
// 			PrivacyProtocol:          gosnmp.AES256, // Use AES for privacy
// 			PrivacyPassphrase:        privPassword,
// 			AuthoritativeEngineTime:  uint32(time.Now().Unix()),
// 		},
// 		Retries: constants.MaxRetries,
// 		Logger:  gosnmp.NewLogger(log.New(os.Stdout, "", 0)),
// 	}

// 	// Connect to the device
// 	err := snmp.Connect()
// 	if err != nil {
// 		log.Println("SNMP Connection Error", err.Error())
// 		return ServiceMonitorStatus{
// 			Name:          config.Name,
// 			Device:        config.Device,
// 			LiveCheckFlag: constants.Escalation,
// 			Status:        "Network Device Connection Error " + err.Error(),
// 			LastCheckTime: time.Now(),
// 			FailureCount:  1,
// 		}, false
// 	}

// 	defer func(Conn net.Conn) {
// 		err := Conn.Close()
// 		if err != nil {
// 			fmt.Printf("Error closing connection: %v", err)
// 		}
// 	}(snmp.Conn)

// 	// Get SNMP metrics
// 	fmt.Printf("Device Information for %s: ---------------- \n", config.Host)
// 	var metrics []mstypes.NetworkDeviceMetric

// 	for _, metric := range commonMetrics {
// 		result, err := snmp.Get([]string{metric.OID})
// 		if err != nil {
// 			log.Printf("Error getting %s: %v ============= \n", metric.Description, err)
// 			continue
// 		}

// 		// Display the result
// 		for _, variable := range result.Variables {
// 			fmt.Printf("OID: %s\n", variable.Name)
// 			switch variable.Type {
// 			case gosnmp.OctetString:
// 				fmt.Printf("Value: %s\n", string(variable.Value.([]byte)))
// 			case gosnmp.TimeTicks:
// 				fmt.Printf("TimeTicks: %d\n", gosnmp.ToBigInt(variable.Value))
// 			case gosnmp.Counter64:
// 				fmt.Printf("Counter: %d\n", variable.Value.(uint64))
// 			default:
// 				fmt.Printf("Value: %v\n", variable.Value)
// 			}
// 		}
// 	}

// 	repository.SyncNetworkMetrics(db, metrics)

// 	// Get interface information
// 	// interfaces, err := getInterfaces(snmp)
// 	// if err != nil {
// 	// 	return ServiceMonitorStatus{
// 	// 		Name:          config.Name,
// 	// 		Device:        config.Device,
// 	// 		LiveCheckFlag: constants.Escalation,
// 	// 		//Status:        "Error getting SNMP interfaces",
// 	// 		Status:        "Error getting SNMP interfaces " + err.Error(),
// 	// 		LastCheckTime: time.Now(),
// 	// 		FailureCount:  1,
// 	// 	}, false
// 	// }

// 	// fmt.Println("\nInterface Information:")
// 	// fmt.Println("----------------------------------------")
// 	// for _, iface := range interfaces {
// 	// 	fmt.Printf("Interface: %s\n", iface)
// 	// }

// 	return ServiceMonitorStatus{
// 		Name:              config.Name,
// 		Device:            config.Device,
// 		LiveCheckFlag:     constants.Healthy,
// 		Status:            "Healthy",
// 		LastCheckTime:     time.Now(),
// 		LastServiceUpTime: time.Now(),
// 		FailureCount:      0,
// 	}, true
// }

// // func getInterfacesV3(snmp *gosnmp.GoSNMP) ([]string, error) {
// // 	var interfaces []string

// // 	//err_ := snmp.Walk("1.3.6.1.2.1.2.2.1", func(variable gosnmp.SnmpPDU) error {
// // 	//	fmt.Printf("OID: %s, Value: %v\n", variable.Name, variable.Value)
// // 	//	return nil
// // 	//})
// // 	//if err_ != nil {
// // 	//	log.Fatalf("Error walking SNMP table: %v", err_)
// // 	//}

// // 	// Get interface descriptions
// // 	// Define the walk function
// // 	walkFn := func(pdu gosnmp.SnmpPDU) error {
// // 		if pdu.Type == gosnmp.OctetString {
// // 			interfaces = append(interfaces, string(pdu.Value.([]byte)))
// // 		}
// // 		return nil
// // 	}

// // 	// Get interface descriptions using BulkWalk with callback
// // 	err := snmp.BulkWalk(".1.3.6.1.2.1.2.2.1.2", walkFn)
// // 	if err != nil {
// // 		return nil, fmt.Errorf("BulkWalk error: %v", err)
// // 	}

// // 	return interfaces, nil
// // }

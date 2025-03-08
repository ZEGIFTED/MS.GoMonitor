package monitors

import (
	"fmt"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
	"github.com/gosnmp/gosnmp"
	"log"
	"net"
	"time"
)

func (service *SNMPServiceChecker) Check(config ServiceMonitorData) (ServiceMonitorStatus, bool) {
	host := config.Host
	port := config.Port

	if host == "" {
		return ServiceMonitorStatus{
			Name:          config.Name,
			Device:        config.Device,
			LiveCheckFlag: constants.Degraded,
			Status:        "Invalid SNMP configuration",
			LastCheckTime: time.Now(),
			FailureCount:  0,
		}, false
	}

	// SNMPMetric represents an SNMP OID and its description
	type SNMPMetric struct {
		OID         string
		Description string
	}

	// Common SNMP OIDs for system information
	var commonMetrics = []SNMPMetric{
		{".1.3.6.1.2.1.1.1.0", "System Description"},
		{".1.3.6.1.2.1.1.3.0", "Uptime"},
		{".1.3.6.1.2.1.1.5.0", "System Name"},
		{".1.3.6.1.2.1.2.1.0", "Number of Interfaces"},
		{".1.3.6.1.2.1.25.2.2.0", "Memory"},
		{".1.3.6.1.2.1.31.1.1.1.6.1", "Inbound traffic"},
		{".1.3.6.1.2.1.31.1.1.1.10.1", "Outbound traffic"},
	}

	//ifInOctetsOID  = "1.3.6.1.2.1.2.2.1.10.1" // Incoming traffic (Replace 1 with your interface index)
	//ifOutOctetsOID = "1.3.6.1.2.1.2.2.1.16.1" // Outgoing traffic

	//".1.3.6.1.2.1.2.2.1.10", // ifInOctets
	//	".1.3.6.1.2.1.2.2.1.16", // ifOutOctets
	//	".1.3.6.1.2.1.2.2.1.5",  // ifSpeed

	CommunityString, ok := config.Configuration["communityString"]

	fmt.Println(CommunityString)
	if !ok || CommunityString == "" {
		log.Println("communityString is missing in configuration... Using default")

		CommunityString = "public"
	}

	community, ok := CommunityString.(string)
	if !ok {
		log.Println("communityString is not a string")
	}

	fmt.Println(host, port, community)
	// Configure SNMP connection
	snmp := &gosnmp.GoSNMP{
		Target:    host, // Replace with your device's IP
		Port:      121,
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(10) * time.Second,
		Retries:   constants.MaxRetries,
	}

	// Connect to the device
	err := snmp.Connect()
	if err != nil {
		return ServiceMonitorStatus{
			Name:          config.Name,
			Device:        config.Device,
			LiveCheckFlag: constants.Escalation,
			Status:        "Error getting SNMP interfaces " + err.Error(),
			LastCheckTime: time.Now(),
			FailureCount:  0,
			//LastErrorLog:  fmt.Sprintf("Error connecting to device: %v", err),
		}, false
	}

	defer func(Conn net.Conn) {
		err := Conn.Close()
		if err != nil {
			fmt.Printf("Error closing connection: %v", err)
		}
	}(snmp.Conn)

	// Get SNMP metrics
	fmt.Printf("Device Information for %s:\n", config.Host)
	fmt.Println("----------------------------------------")

	for _, metric := range commonMetrics {
		result, err := snmp.Get([]string{metric.OID})
		if err != nil {
			log.Printf("Error getting %s: %v\n", metric.Description, err)
			continue
		}

		// Display the result
		for _, variable := range result.Variables {
			fmt.Printf("OID: %s\n", variable.Name)
			switch variable.Type {
			case gosnmp.OctetString:
				fmt.Printf("Value: %s\n", string(variable.Value.([]byte)))
			case gosnmp.TimeTicks:
				fmt.Printf("TimeTicks: %d\n", gosnmp.ToBigInt(variable.Value))
			case gosnmp.Counter64:
				fmt.Printf("Counter: %d\n", variable.Value.(uint64))
			default:
				fmt.Printf("Value: %v\n", variable.Value)
			}
		}
	}

	// Get interface information
	interfaces, err := getInterfaces(snmp)
	if err != nil {
		return ServiceMonitorStatus{
			Name:          config.Name,
			Device:        config.Device,
			LiveCheckFlag: constants.Escalation,
			//Status:        "Error getting SNMP interfaces",
			Status:        "Error getting SNMP interfaces " + err.Error(),
			LastCheckTime: time.Now(),
			FailureCount:  1,
		}, false
	}

	fmt.Println("\nInterface Information:")
	fmt.Println("----------------------------------------")
	for _, iface := range interfaces {
		fmt.Printf("Interface: %s\n", iface)
	}

	return ServiceMonitorStatus{
		Name:              config.Name,
		Device:            config.Device,
		LiveCheckFlag:     constants.Healthy,
		Status:            "Healthy",
		LastCheckTime:     time.Now(),
		LastServiceUpTime: time.Now(),
		FailureCount:      0,
	}, true
}

func getInterfaces(snmp *gosnmp.GoSNMP) ([]string, error) {
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
		if pdu.Type == gosnmp.OctetString {
			interfaces = append(interfaces, string(pdu.Value.([]byte)))
		}
		return nil
	}

	// Get interface descriptions using BulkWalk with callback
	err := snmp.BulkWalk(".1.3.6.1.2.1.2.2.1.2", walkFn)
	if err != nil {
		return nil, fmt.Errorf("BulkWalk error: %v", err)
	}

	return interfaces, nil
}

package monitors

import (
	"fmt"
	"log"
	"net"

	// "os"
	"strconv"
	"strings"
	"time"

	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
	"github.com/gosnmp/gosnmp"
)

type NetworkManager struct {
	snmp         *gosnmp.GoSNMP
	trap         *gosnmp.TrapListener
	Target       string
	Community    string
	TrapPort     int
	Username     string
	AuthPassword string
	PrivPassword string
}

func (nm *NetworkManager) NewNM(device string) *NetworkManager {
	var snmp gosnmp.GoSNMP

	if strings.EqualFold(string(device), "NetworkV2") {
		log.Println("Using SNMP v2")

		snmp = gosnmp.GoSNMP{
			Target:    nm.Target, // Replace with your device's IP
			Port:      161,
			Community: nm.Community,
			Version:   gosnmp.Version2c,
			Timeout:   time.Duration(15) * time.Second,
			Retries:   constants.MaxRetries,
			// Logger:  gosnmp.NewLogger(log.New(os.Stdout, "", 0)),
		}
	} else {
		snmp = gosnmp.GoSNMP{
			Target:        nm.Target,
			Port:          uint16(nm.TrapPort),
			Version:       gosnmp.Version3,
			Timeout:       time.Duration(35) * time.Second,
			SecurityModel: gosnmp.Default.SecurityModel,
			MsgFlags:      gosnmp.Default.MsgFlags, // Use AuthPriv for both authentication and privacy
			SecurityParameters: &gosnmp.UsmSecurityParameters{
				UserName:                 nm.Username,
				AuthenticationProtocol:   gosnmp.SHA512, // Use SHA for authentication
				AuthenticationPassphrase: nm.AuthPassword,
				PrivacyProtocol:          gosnmp.AES256, // Use AES for privacy
				PrivacyPassphrase:        nm.PrivPassword,
				AuthoritativeEngineTime:  uint32(time.Now().Unix()),
			},
			Retries: constants.MaxRetries,
			// Logger:  gosnmp.NewLogger(log.New(os.Stdout, "", 0)),
		}
	}

	return &NetworkManager{
		snmp: &snmp,
	}
}

func (nm *NetworkManager) GetFormattedSNMPValue(result []gosnmp.SnmpPDU) (string, string) {
	for _, variable := range result {
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

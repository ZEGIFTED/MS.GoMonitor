package network

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type Tracer struct {
	destination string
}

func NewTracer(destination string) *Tracer {
	return &Tracer{destination: destination}
}

func (t *Tracer) Traceroute() {
	ipAddr, err := net.ResolveIPAddr("ip4", t.destination)
	if err != nil {
		fmt.Println("Failed to resolve destination:", err)
		return
	}

	fmt.Printf("Traceroute to %s (%s)\n", t.destination, ipAddr)

	for ttl := 1; ttl <= 20; ttl++ {
		conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
		if err != nil {
			fmt.Println("Failed to listen for ICMP packets:", err)
			return
		}
		defer conn.Close()

		conn.IPv4PacketConn().SetTTL(ttl)

		msg := icmp.Message{
			Type: ipv4.ICMPTypeEcho, Code: 0,
			Body: &icmp.Echo{
				ID: os.Getpid() & 0xffff, Seq: ttl,
				Data: []byte("HELLO"),
			},
		}

		msgBytes, err := msg.Marshal(nil)
		if err != nil {
			fmt.Println("Failed to marshal ICMP message:", err)
			return
		}

		start := time.Now()
		if _, err := conn.WriteTo(msgBytes, ipAddr); err != nil {
			fmt.Println("Failed to send ICMP message:", err)
			return
		}

		reply := make([]byte, 1500)
		if err := conn.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
			fmt.Println("Failed to set read deadline:", err)
			return
		}

		n, peer, err := conn.ReadFrom(reply)
		if err != nil {
			fmt.Printf("%d\t*\n", ttl)
			continue
		}

		duration := time.Since(start)
		rm, err := icmp.ParseMessage(1, reply[:n])
		if err != nil {
			fmt.Println("Failed to parse ICMP message:", err)
			return
		}

		switch rm.Type {
		case ipv4.ICMPTypeTimeExceeded:
			fmt.Printf("%d\t%v\t%v\n", ttl, peer, duration)
		case ipv4.ICMPTypeEchoReply:
			fmt.Printf("%d\t%v\t%v\n", ttl, peer, duration)
			return
		default:
			fmt.Printf("%d\t%v\t%v\tunexpected type %v\n", ttl, peer, duration, rm.Type)
		}
	}
}

func main() {
	tracer := NewTracer("www.google.com")
	tracer.Traceroute()
}

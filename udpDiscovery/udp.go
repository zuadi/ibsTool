package udpDiscovery

import (
	"fmt"
	"ibsTool/logging"
	"net"
	"os"
	"strings"
	"time"
)

var webPort = "8080" // Your web server port

const (
	udpPort     = ":9999"
	discoverKey = "LUCKFOX_DISCOVER"
)

func UPDListener(l *logging.Logger) error {
	fmt.Println(1)
	if p := os.Getenv("PORT"); p != "" {
		webPort = ":" + p
	}

	addr, err := net.ResolveUDPAddr("udp", udpPort)
	if err != nil {
		return fmt.Errorf("error resolving udp address: %v\n", err)

	}
	fmt.Println(2)

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("error starting udp listener: %v\n", err)
	}
	fmt.Println(3)

	defer conn.Close()
	fmt.Println(4)

	l.BroadcastLog(fmt.Sprintf("luckfox udp discovery listening on %s...\n", udpPort))

	buf := make([]byte, 1024)
	fmt.Println(5)

	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		message := strings.TrimSpace(string(buf[:n]))

		// Check if the received message matches our discovery key
		if message == discoverKey {
			hostname, _ := os.Hostname()
			localIP := getLocalIP()

			// Prepare response payload
			response := fmt.Sprintf("LUCKFOX_RESPONSE|Hostname:%s|IP:%s|WebPort:%s", hostname, localIP, webPort)

			_, err = conn.WriteToUDP([]byte(response), remoteAddr)
			if err == nil {
				l.BroadcastLog(fmt.Sprintf("responded to discovery request from %s\n", remoteAddr.String()))
			}
		}
	}
	fmt.Println(6)
	return nil

}

func DiscoverDevices() (foundIp []string, err error) {
	broadcastAddr, err := net.ResolveUDPAddr("udp", "255.255.255.255"+udpPort)
	if err != nil {
		return nil, fmt.Errorf("Resolve failed: %v\n", err)

	}

	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, fmt.Errorf("Listen failed: %v\n", err)

	}
	defer conn.Close()

	// Send discovery broadcast
	message := []byte("LUCKFOX_DISCOVER")
	_, err = conn.WriteTo(message, broadcastAddr)
	if err != nil {
		return nil, fmt.Errorf("Broadcast failed: %v\n", err)
	}

	fmt.Println("Broadcasting for Luckfox devices on local network...")

	// Set timeout for responses (3 seconds)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	buf := make([]byte, 1024)

	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			// Timeout reached
			break
		}

		fmt.Printf("Found device at %s -> %s\n", remoteAddr.IP.String(), string(buf[:n]))
		foundIp = append(foundIp, remoteAddr.IP.String())
	}

	return
}

// Helper to get non-loopback IPv4 address
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "unknown"
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "unknown"
}

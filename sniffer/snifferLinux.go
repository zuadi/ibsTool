//go:build linux

package sniffer

import (
	"encoding/json"
	"log"
	"net"
	"syscall"
	"time"

	"ibsTool/logging"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	wsModels "github.com/zuadi/webServer/models"
)

var jsonMarshal = json.Marshal

// htons converts uint16 from host byte order to network byte order
func htons(i uint16) uint16 {
	return (i<<8)&0xff00 | (i>>8)&0x00ff
}

func StartTCP(ws *wsModels.WSClient, l *logging.Logger) {
	// ETH_P_ALL = 0x0003 (Capture all Ethernet frames)
	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(htons(0x0003)))
	if err != nil {
		log.Fatalf("Failed to open AF_PACKET raw socket on Linux: %v", err)
	}
	defer syscall.Close(fd)

	// Bind socket to specific interface if available (e.g. eth0)
	iface, err := net.InterfaceByName("eth0")
	if err == nil {
		addr := &syscall.SockaddrLinklayer{
			Protocol: htons(0x0003),
			Ifindex:  iface.Index,
		}
		if err := syscall.Bind(fd, addr); err != nil {
			log.Fatalf("Failed to bind raw socket to interface eth0: %v", err)
		}
	}

	if l != nil {
		l.BroadcastLog("⚡ Linux Raw Socket sniffer active on port 502...")
	}

	buf := make([]byte, 65536)
	for {
		// syscall.Read returns exactly 2 values: (n, err)
		n, err := syscall.Read(fd, buf)
		if err != nil || n <= 0 {
			continue
		}

		packetData := buf[:n]
		packet := gopacket.NewPacket(packetData, layers.LayerTypeEthernet, gopacket.Default)

		ipLayer := packet.Layer(layers.LayerTypeIPv4)
		tcpLayer := packet.Layer(layers.LayerTypeTCP)
		if ipLayer == nil || tcpLayer == nil {
			continue
		}

		tcp, _ := tcpLayer.(*layers.TCP)
		if tcp.SrcPort != 502 && tcp.DstPort != 502 {
			continue
		}

		ip, _ := ipLayer.(*layers.IPv4)
		payload := tcp.Payload
		if len(payload) == 0 {
			continue
		}

		statsMutex.Lock()
		processPayload(payload, ip.SrcIP.String(), ip.DstIP.String(), tcp.DstPort == 502, time.Now(), ws)
		statsMutex.Unlock()
	}
}

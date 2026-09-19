//go:build windows

package sniffer

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ibsTool/logging"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"

	wsModels "github.com/zuadi/webServer/models"
)

var jsonMarshal = json.Marshal

func StartTCP(ws *wsModels.WSClient, l *logging.Logger) {
	var handle *pcap.Handle
	var err error
	var selectedDevice string

	devices, devErr := pcap.FindAllDevs()
	if l != nil {
		for i, o := range devices {
			l.BroadcastLog(fmt.Sprintf("options %d %s %s", i, o.Description, o.Name))
		}
	}
	if devErr != nil || len(devices) == 0 {
		log.Fatalf("Critical: No active network interfaces found. Ensure Npcap is installed: %v", devErr)
	}

	for i, device := range devices {
		if i != 8 {
			continue
		}
		if len(device.Addresses) > 0 {
			selectedDevice = device.Name
			log.Printf("Selected network interface: %s (%s)", device.Description, device.Name)
			break
		}
	}

	if selectedDevice == "" {
		selectedDevice = devices[0].Name
		log.Printf("Fallback: Selected network interface: %s", devices[0].Description)
	}

	handle, err = pcap.OpenLive(selectedDevice, 65536, true, pcap.BlockForever)
	if err != nil {
		log.Fatalf("Failed to open Windows device %s: %v", selectedDevice, err)
	}
	defer handle.Close()

	if err := handle.SetBPFFilter("tcp port 502"); err != nil {
		log.Fatalf("BPF Filter Error: %v", err)
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	l.BroadcastLog("⚡ Windows Packet sniffer routing active on port 502...")

	for packet := range packetSource.Packets() {
		ipLayer := packet.Layer(layers.LayerTypeIPv4)
		tcpLayer := packet.Layer(layers.LayerTypeTCP)
		if ipLayer == nil || tcpLayer == nil {
			continue
		}

		ip, _ := ipLayer.(*layers.IPv4)
		tcp, _ := tcpLayer.(*layers.TCP)
		payload := tcp.Payload
		if len(payload) == 0 {
			continue
		}

		statsMutex.Lock()
		processPayload(payload, ip.SrcIP.String(), ip.DstIP.String(), tcp.DstPort == 502, time.Now(), ws)
		statsMutex.Unlock()
	}
}

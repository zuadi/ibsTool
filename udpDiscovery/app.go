package udpdiscovery

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Device struct {
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
	Port     string `json:"port"`
	URL      string `json:"url"`
}

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

// DiscoverDevices sends a UDP broadcast and gathers responding INS-Tool devices
func (a *App) DiscoverDevices() ([]Device, error) {
	broadcastAddr, err := net.ResolveUDPAddr("udp", "255.255.255.255:9999")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve broadcast: %w", err)
	}

	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to listen UDP: %w", err)
	}
	defer conn.Close()

	// 2-second timeout window for responses
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	// Send discovery key
	payload := []byte("LUCKFOX_DISCOVER")
	_, err = conn.WriteTo(payload, broadcastAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to send discovery: %w", err)
	}

	devices := []Device{}
	buf := make([]byte, 1024)

	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			break // Timeout or end of discovery
		}

		rawResponse := string(buf[:n])
		// Expected format: LUCKFOX_RESPONSE|Hostname:panel1|IP:192.168.1.50|WebPort:8080
		if strings.HasPrefix(rawResponse, "LUCKFOX_RESPONSE") {
			parts := strings.Split(rawResponse, "|")
			device := Device{IP: remoteAddr.IP.String(), Port: "8080"}

			for _, part := range parts[1:] {
				kv := strings.SplitN(part, ":", 2)
				if len(kv) == 2 {
					switch kv[0] {
					case "Hostname":
						device.Hostname = kv[1]
					case "IP":
						if kv[1] != "" {
							device.IP = kv[1]
						}
					case "WebPort":
						device.Port = kv[1]
					}
				}
			}

			device.URL = fmt.Sprintf("http://%s%s", device.IP, device.Port)
			devices = append(devices, device)
		}
	}

	return devices, nil
}

// OpenInBrowser opens the selected URL in the OS default web browser
func (a *App) OpenInBrowser(urlStr string) {
	// 1. Ensure scheme is present
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "http://" + urlStr
	}

	// 2. Fix double colons or malformed IP:port strings if present
	urlStr = strings.ReplaceAll(urlStr, "::", ":")

	// 3. Open in OS default browser
	runtime.BrowserOpenURL(a.ctx, urlStr)
}

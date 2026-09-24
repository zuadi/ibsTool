package utils

import (
	"bytes"
	"errors"
	"fmt"
	"ibsTool/models"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"go.bug.st/serial/enumerator"
)

func GetSerialPorts() (ports []*enumerator.PortDetails, err error) {
	// Get basic port name list (e.g., "COM1", "/dev/ttyUSB0")
	detailedPorts, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, fmt.Errorf("Error enumerating ports: %v", err)
	}

	var activePorts []*enumerator.PortDetails
	for _, p := range detailedPorts {
		// Optional: Filter out raw ttyS ports if you only target USB devices
		// if strings.HasPrefix(p.Name, "/dev/ttyS") && !p.IsUSB { continue }
		activePorts = append(activePorts, p)
	}

	if len(activePorts) == 0 {
		return nil, errors.New("no serial ports found")
	}

	return activePorts, nil
}

func GetActiveEthernet() (*models.ActiveAdapter, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate interfaces: %w", err)
	}

	for _, iface := range interfaces {
		// 1. Must be UP and running (cable connected)
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// 2. Filter out Wi-Fi or virtual interfaces by standard naming conventions
		name := strings.ToLower(iface.Name)
		if strings.Contains(name, "wlan") || strings.Contains(name, "wi-fi") ||
			strings.Contains(name, "docker") || strings.Contains(name, "veth") ||
			strings.Contains(name, "tailscale") || strings.Contains(name, "tun") || strings.Contains(name, "vmnet") {
			continue
		}

		// 3. Inspect interface IP addresses
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.IsLoopback() {
				continue
			}

			// Return the first valid IPv4 on an active physical link
			if ipv4 := ipNet.IP.To4(); ipv4 != nil {
				return &models.ActiveAdapter{
					Name:       iface.Name,
					IP:         ipv4.String(),
					SubnetMask: fmt.Sprintf("%d.%d.%d.%d", ipNet.Mask[0], ipNet.Mask[1], ipNet.Mask[2], ipNet.Mask[3]),
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("no active physical ethernet connection found")
}

func GetActiveInterfaceByName(key string) (*models.ActiveAdapter, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate interfaces: %w", err)
	}

	for _, iface := range interfaces {
		// 1. Must be UP and running (cable connected)
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// 2. Filter out Wi-Fi or virtual interfaces by standard naming conventions
		name := strings.ToLower(iface.Name)

		if !strings.Contains(name, key) {
			continue
		}

		// 3. Inspect interface IP addresses
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.IsLoopback() {
				continue
			}

			// Return the first valid IPv4 on an active physical link
			if ipv4 := ipNet.IP.To4(); ipv4 != nil {
				adapter := &models.ActiveAdapter{
					Name:       iface.Name,
					IP:         ipv4.String(),
					SubnetMask: fmt.Sprintf("%d.%d.%d.%d", ipNet.Mask[0], ipNet.Mask[1], ipNet.Mask[2], ipNet.Mask[3]),
				}

				if wifi, err := GetWifiStats(iface.Name); err == nil {
					adapter.SSID = wifi.SSID
					adapter.Signal = wifi.Signal
					adapter.Channel = wifi.Channel
				}
				return adapter, err
			}
		}
	}
	return nil, fmt.Errorf("no active physical ethernet connection found")
}

func GetWifiStats(interfaceName string) (status models.ActiveAdapter, err error) {
	status.Name = interfaceName

	// 1. Get Subnet information via standard library
	if iface, err := net.InterfaceByName(interfaceName); err == nil {
		if addrs, err := iface.Addrs(); err == nil {
			for _, addr := range addrs {
				if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
					mask := ipNet.Mask
					status.SubnetMask = fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
					break
				}
			}
		}
	}

	// 2. Query OS Wi-Fi information
	switch runtime.GOOS {
	case "windows":
		// Wrap interfaceName in quotes to handle spaces (e.g., name="Wi-Fi")
		cmd := exec.Command("netsh", "wlan", "show", "interfaces", fmt.Sprintf("name=\"%s\"", interfaceName))
		var out bytes.Buffer
		cmd.Stdout = &out

		if err := cmd.Run(); err != nil || out.Len() == 0 {
			// Fallback: Query all interfaces if specific name lookup fails
			out.Reset()
			cmd = exec.Command("netsh", "wlan", "show", "interfaces")
			cmd.Stdout = &out
			if err := cmd.Run(); err != nil {
				return status, fmt.Errorf("netsh command failed: %w", err)
			}
		}

		lines := strings.Split(out.String(), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)

			key, value, found := strings.Cut(line, ":")
			if !found {
				continue
			}

			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)

			// Strict equality checks to avoid matching BSSID or Profile lines
			switch key {
			case "SSID":
				status.SSID = value
			case "Signal":
				status.Signal = value // e.g., "85%"
			case "Channel":
				status.Channel = value
			}
		}

	default: // Linux
		cmd := exec.Command("iw", "dev", interfaceName, "link")
		var out bytes.Buffer
		cmd.Stdout = &out

		if err := cmd.Run(); err != nil {
			return status, fmt.Errorf("iw command failed for interface %s: %w", interfaceName, err)
		}

		lines := strings.Split(out.String(), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)

			if strings.HasPrefix(line, "SSID:") {
				status.SSID = strings.TrimSpace(strings.TrimPrefix(line, "SSID:"))
			} else if strings.HasPrefix(line, "signal:") {
				status.Signal = strings.TrimSpace(strings.TrimPrefix(line, "signal:"))
			} else if strings.HasPrefix(line, "freq:") {
				freqStr := strings.TrimSpace(strings.TrimPrefix(line, "freq:"))
				if freq, err := strconv.Atoi(freqStr); err == nil {
					status.Channel = strconv.Itoa(freqToChannel(freq))
				} else {
					status.Channel = freqStr
				}
			}
		}
	}
	return status, nil
}

// Convert 802.11 frequency (MHz) to channel number
func freqToChannel(freq int) int {
	switch {
	case freq >= 2412 && freq <= 2484:
		return (freq-2412)/5 + 1
	case freq >= 5180 && freq <= 5825:
		return (freq-5180)/5 + 36
	default:
		return 0
	}
}

// GetTailscaleAdapter fetches the IP for tailscale0
func GetTailscaleAdapter() (*models.ActiveAdapter, error) {
	return GetActiveInterfaceByName("tailscale")
}

// GetHostapdStations parses connected clients from hostapd control interface
func GetHostapdStations() ([]models.Station, error) {
	cmd := exec.Command("hostapd_cli", "all_sta")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	var stations []models.Station
	var current *models.Station

	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, ":") && len(line) == 17 && !strings.Contains(line, "=") {
			// New MAC address block starts
			if current != nil {
				stations = append(stations, *current)
			}
			current = &models.Station{Mac: line}
			continue
		}

		if current != nil {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key, val := parts[0], parts[1]
				switch key {
				case "signal":
					if rssi, err := strconv.Atoi(val); err == nil {
						current.Rssi = rssi
					}
				case "connected_time":
					if secs, err := strconv.Atoi(val); err == nil {
						hours := secs / 3600
						mins := (secs % 3600) / 60
						current.ConnectedTime = strconv.Itoa(hours) + "h " + strconv.Itoa(mins) + "m"
					}
				}
			}
		}
	}
	if current != nil {
		stations = append(stations, *current)
	}

	return stations, nil
}

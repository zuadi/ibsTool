package models

type System struct {
	Lan          *ActiveAdapter `json:"lan,omitempty"`
	Wlan         *ActiveAdapter `json:"wlan,omitempty"`
	Tailscale    *ActiveAdapter `json:"tailscale,omitempty"`
	SSID         string         `json:"ssid,omitempty"`
	APChannel    string         `json:"ap_channel,omitempty"`
	ClientsCount int            `json:"clients_count,omitempty"`
	CPU          float64        `json:"cpu,omitempty"`
	RAM          float64        `json:"ram,omitempty"`
	Disk         float64        `json:"disk,omitempty"`
	Stations     []Station      `json:"stations,omitempty"`
}

type Station struct {
	Mac           string `json:"mac,omitempty"`
	Ip            string `json:"ip,omitempty"`
	Rssi          int    `json:"rssi,omitempty"`
	TXRate        string `json:"tx_rate,omitempty"`
	ConnectedTime string `json:"connected_time,omitempty"`
}

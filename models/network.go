package models

type ActiveAdapter struct {
	Name       string `json:"name"`
	IP         string `json:"ip"`
	SSID       string `json:"ssid,omitempty"`
	SubnetMask string `json:"subnet,omitempty"`
	Signal     string `json:"signal,omitempty"`
	Channel    string `json:"channel,omitempty"`
}

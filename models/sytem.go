package models

type System struct {
	Lan       *ActiveAdapter `json:"lan,omitempty"`
	Wlan      *ActiveAdapter `json:"wlan,omitempty"`
	Tailscale *ActiveAdapter `json:"tailscale,omitempty"`
}

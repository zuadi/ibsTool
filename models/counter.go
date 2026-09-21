package models

type Counter struct {
	Requests   int `json:"requests"`
	Response   int `json:"responses"`
	Exceptions int `json:"exceptions"`
}

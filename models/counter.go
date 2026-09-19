package models

type Counter struct {
	Requests  int `json:"requests"`
	Response  int `json:"response"`
	Exeptions int `json:"exeptions"`
}

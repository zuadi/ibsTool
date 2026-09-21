package models

type Settings struct {
	Service    string `json:"service" yaml:"-"`
	Action     string `json:"action" yaml:"-"`
	Port       string `json:"port" yaml:"port"`
	DecodeMode string `json:"decodeMode" yaml:"decodeMode"`
	BaudRate   int    `json:"baudrate" yaml:"baudrate"`
	Databits   int    `json:"databits" yaml:"databits"`
	Parity     string `json:"parity" yaml:"parity"`
	Stopbits   int    `json:"stopbits" yaml:"stopbits"`
}

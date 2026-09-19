package models

type Settings struct {
	Action     string `json:"action,omitempty" yaml:"-"`
	Port       string `json:"port,omitempty" yaml:"port"`
	DecodeMode string `json:"decodeMode,omitempty" yaml:"decodeMode"`
	BaudRate   int    `json:"baudrate,omitempty" yaml:"baudrate"`
	Databits   int    `json:"databits,omitempty" yaml:"databits"`
	Parity     string `json:"parity,omitempty" yaml:"parity"`
	Stopbits   int    `json:"stopbits,omitempty" yaml:"stopbits"`
}

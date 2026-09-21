package models

type ModbusFrame struct {
	Timestamp   string   `json:"timestamp"`
	Source      string   `json:"source"`
	Destination string   `json:"destination"`
	Hex         string   `json:"hex"`
	BytesList   []string `json:"bytes_list"`
	RawInts     []int    `json:"raw_ints"`
	Integers    []int    `json:"integers"`
	Type        string   `json:"type"`
	FC          string   `json:"fc"`
	Latency     string   `json:"latency"`
	Exception   string   `json:"exception"`
}

type ModbusRTUFrame struct {
	Service        string   `json:"service"`
	Timestamp      string   `json:"timestamp"`
	Hex            string   `json:"hex"`
	BytesList      []string `json:"bytes_list"`
	RawInts        []int    `json:"raw_ints"`
	SlaveID        int      `json:"slave_id"`
	Type           string   `json:"type"`
	Source         string   `json:"source"`
	Destination    string   `json:"destination"`
	FC             string   `json:"fc"`
	CRC            string   `json:"crc"`
	IsValidCRC     bool     `json:"is_valid_crc"`
	DecodedPayload []int16  `json:"decoded_payload"`
	Counter        *Counter `json:"counter"`
}

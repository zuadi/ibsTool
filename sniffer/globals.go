package sniffer

// Parity describes a serial port parity setting
type Parity int

const (
	// NoParity disable parity control (default)
	NoParity Parity = iota
	// OddParity enable odd-parity check
	OddParity
	// EvenParity enable even-parity check
	EvenParity
	// MarkParity enable mark-parity (always 1) check
	MarkParity
	// SpaceParity enable space-parity (always 0) check
	SpaceParity
)

func (p *Parity) SetParity(s string) {
	switch s {
	case "None":
		*p = NoParity
	case "Odd":
		*p = OddParity
	case "Even":
		*p = EvenParity
	case "Mark":
		*p = MarkParity
	case "Space":
		*p = SpaceParity
	}
}

func (p *Parity) GetParity() string {
	switch *p {
	case NoParity:
		return "None"
	case OddParity:
		return "Odd"
	case EvenParity:
		return "Even"
	case MarkParity:
		return "Mark"
	case SpaceParity:
		return "Space"
	}
	return ""
}

// StopBits describe a serial port stop bits setting
type StopBits int

const (
	// OneStopBit sets 1 stop bit (default)
	OneStopBit StopBits = iota
	// OnePointFiveStopBits sets 1.5 stop bits
	OnePointFiveStopBits
	// TwoStopBits sets 2 stop bits
	TwoStopBits
)

package sniffer

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"ibsTool/logging"
	"ibsTool/models"

	wsModels "github.com/zuadi/webServer/models"
	"go.bug.st/serial"
)

const (
	RTU        = "rtu"
	INIT       = "init"
	CONNECT    = "connect"
	DISCONNECT = "disconnect"
	CONFIG     = "config"
	RESETSTAT  = "resetstat"
)

type ModbusRTUSniffer struct {
	simulation      bool
	serialPort      serial.Port
	webSocket       *wsModels.WSClient
	timeout         time.Duration
	logger          *logging.Logger
	counter         *models.Counter
	config          *ConfigHandler
	state           string
	cancel          context.CancelFunc
	mu              sync.RWMutex
	lastRequestTime time.Time
	latence         float64
}

func NewModbusRTUSniffer(ws *wsModels.WSClient, l *logging.Logger) (*ModbusRTUSniffer, error) {
	sniffer := &ModbusRTUSniffer{
		webSocket: ws,
		timeout:   5 * time.Millisecond,
		state:     DISCONNECT,
		logger:    l,
	}

	configPath := "./config/modbusRTU.yml"
	absolutePath, _ := filepath.Abs(configPath)
	l.BroadcastLog("attempt to open config: " + absolutePath)
	sniffer.config = NewConfigHandler(configPath)
	settings, err := sniffer.config.ReadConfig()
	if err != nil {
		l.BroadcastLog("config file '" + absolutePath + "' not found use default")
	}

	ws.NewConnection = func(wsClient *wsModels.WSClient) {
		sniffer.mu.RLock()
		defer sniffer.mu.RUnlock()
		if settings == nil {
			wsClient.Answer(wsModels.TextMessage, models.SetServiceAction(RTU, sniffer.state))
			return
		}

		settings.Service = RTU
		settings.Action = INIT
		settings.State = sniffer.state

		b, err := json.Marshal(settings)
		if err != nil {
			sniffer.logger.BroadcastLog(err)
			return
		}
		wsClient.Answer(wsModels.TextMessage, b)
	}
	ws.Listen(func(data any) {
		d, ok := data.([]byte)
		if !ok {
			sniffer.logger.BroadcastLog("error data conversion from interface to []byte")
			return
		}

		if err := json.Unmarshal(d, &settings); err != nil {
			sniffer.logger.BroadcastLog(err)
			return
		}

		if settings.Service != RTU {
			return
		}

		var parity Parity
		parity.SetParity(settings.Parity)

		switch settings.Action {
		case DISCONNECT:
			sniffer.logger.BroadcastLog("disconnecting modbus rtu sniffer")
			if err := sniffer.Stop(); err != nil {
				sniffer.logger.BroadcastLog(err)
				ws.Broadcast(wsModels.TextMessage, models.SetServiceError(RTU, err))
			}
		case CONNECT:
			sniffer.logger.BroadcastLog("connecting modbus rtu sniffer")
			go func() {
				if err := sniffer.Start(settings.Port, settings.DecodeMode, settings.BaudRate, parity, settings.Databits, StopBits(settings.Stopbits)); err != nil {
					sniffer.logger.BroadcastLog(err)
					ws.Broadcast(wsModels.TextMessage, models.SetServiceError(RTU, err))
				}
			}()
		case CONFIG:
			sniffer.logger.BroadcastLog("changing configuration for modbus rtu sniffer")
			go func() {
				if err := sniffer.Stop(); err != nil {
					sniffer.logger.BroadcastLog(err)
					ws.Broadcast(wsModels.TextMessage, models.SetServiceError(RTU, err))
				}

				time.Sleep(200 * time.Millisecond)

				if err := sniffer.Start(settings.Port, settings.DecodeMode, settings.BaudRate, parity, settings.Databits, StopBits(settings.Stopbits)); err != nil {
					sniffer.logger.BroadcastLog(err)
					ws.Broadcast(wsModels.TextMessage, models.SetServiceError(RTU, err))
				}
			}()
		case RESETSTAT:
			sniffer.counter = &models.Counter{}
			sniffer.logger.BroadcastLog("reset statistic")
		default:
			sniffer.logger.BroadcastLog("modbus rtu sniffer action '" + settings.Action + "' not supported")
		}
	})

	return sniffer, nil
}

func (rtu *ModbusRTUSniffer) Start(portName string, decodeMode string, baudRate int, parity Parity, dataBit int, stopBit StopBits) error {

	rtu.simulation = os.Getenv("MODBUS_RTU_SIMULATION") == "TRUE"

	rtu.mu.Lock()
	if rtu.state == CONNECT {
		rtu.mu.Unlock()
		return fmt.Errorf("sniffer is already running")
	}

	rtu.counter = &models.Counter{}

	mode := &serial.Mode{
		BaudRate: baudRate,
		DataBits: dataBit,
		Parity:   serial.Parity(parity),
		StopBits: serial.StopBits(stopBit),
	}

	rtu.config.WriteConfig(models.Settings{
		Port:       portName,
		DecodeMode: decodeMode,
		BaudRate:   baudRate,
		Databits:   dataBit,
		Parity:     parity.GetParity(),
		Stopbits:   int(stopBit),
	})

	var serverStream *io.PipeReader
	var clientStream *io.PipeWriter

	if rtu.simulation {
		serverStream, clientStream = io.Pipe()
		go func() {
			for {
				// Sende Request
				req := simulationRequest()
				clientStream.Write(req)
				time.Sleep(50 * time.Millisecond)

				// Sende Response mitzählender Variable
				resp := simulationResponse()
				clientStream.Write(resp)
				time.Sleep(1 * time.Second)
			}
		}()
	} else {
		sp, err := serial.Open(portName, mode)
		if err != nil {
			rtu.mu.Unlock()
			return fmt.Errorf("failed to open serial port %s: %v", portName, err)
		}

		rtu.serialPort = sp
		rtu.serialPort.SetReadTimeout(rtu.timeout)
	}

	ctx, cancel := context.WithCancel(context.Background())
	rtu.cancel = cancel
	rtu.state = CONNECT
	rtu.mu.Unlock()

	rtu.webSocket.Broadcast(wsModels.TextMessage, models.SetServiceAction(RTU, CONNECT))
	rtu.logger.BroadcastLog(fmt.Sprintf("⚡ Sniffing Modbus RTU line on %s at %d baud...", portName, baudRate))

	defer func() {
		rtu.mu.Lock()
		if !rtu.simulation && rtu.serialPort != nil {
			rtu.serialPort.Close()
			rtu.serialPort = nil
		}

		rtu.state = DISCONNECT
		rtu.cancel = nil
		rtu.mu.Unlock()

		rtu.webSocket.Broadcast(wsModels.TextMessage, models.SetServiceAction(RTU, DISCONNECT))
	}()

	buf := make([]byte, 256)
	var frameBuffer []byte

	// Calculate dynamic 3.5 character silence timeout for Modbus RTU
	silenceMs := 38500 / baudRate
	interCharTimeout := max(time.Duration(silenceMs)*time.Millisecond, 4*time.Millisecond)

	idleTimer := time.NewTimer(0)
	if !idleTimer.Stop() {
		select {
		case <-idleTimer.C:
		default:
		}
	}

	for {

		select {
		case <-ctx.Done():
			return nil

		case <-idleTimer.C:
			// Silence detected: flush whatever leftover fragments remain
			if len(frameBuffer) > 0 {
				consumed, validFrame := extractNextModbusFrame(frameBuffer)
				if validFrame != nil {
					rtu.processRTUFrame(validFrame)
					frameBuffer = frameBuffer[consumed:]
				} else {
					// Drop bad/unparseable leftover data after timeout
					frameBuffer = nil
				}
			}

		default:
			var n int
			var err error

			if rtu.simulation {
				n, err = serverStream.Read(buf)
			} else {
				n, err = rtu.serialPort.Read(buf)
			}

			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				time.Sleep(5 * time.Microsecond)
				continue
			}

			if n > 0 {
				frameBuffer = append(frameBuffer, buf[:n]...)

				for len(frameBuffer) >= 4 {

					consumed, validFrame := extractNextModbusFrame(frameBuffer)

					if consumed == 0 {
						break // Not enough bytes yet, wait for more data from serial port
					}

					if validFrame != nil {
						rtu.processRTUFrame(validFrame)
					}

					// Safely advance the buffer whether it was a full frame (consumed > 1)
					// or a 1-byte sliding re-sync (consumed == 1)
					frameBuffer = frameBuffer[consumed:]
				}

				if !idleTimer.Stop() {
					select {
					case <-idleTimer.C:
					default:
					}
				}
				idleTimer.Reset(interCharTimeout)
			}
		}
	}
}

func (rtu *ModbusRTUSniffer) Stop() error {
	rtu.mu.Lock()
	defer rtu.mu.Unlock()

	if rtu.cancel != nil {
		rtu.cancel()
	}

	if !rtu.simulation && rtu.serialPort != nil {
		err := rtu.serialPort.Close()
		rtu.serialPort = nil
		return err
	}

	return nil
}

func (rtu *ModbusRTUSniffer) SetTimeout(t time.Duration) {
	rtu.mu.Lock()
	defer rtu.mu.Unlock()

	rtu.timeout = t
	if !rtu.simulation && rtu.serialPort != nil {
		rtu.serialPort.SetReadTimeout(t)
	}
}

func extractNextModbusFrame(buf []byte) (int, []byte) {
	if len(buf) < 4 {
		return 0, nil
	}

	slaveID := buf[0]
	// Modbus standard valid station addresses are 1 to 247 (0 is broadcast)
	// If the first byte is out of range, slide the window immediately.
	if slaveID == 0 || slaveID > 247 {
		return 1, nil
	}

	funcCode := buf[1]
	expectedLen := 0

	switch funcCode {
	case 0x01, 0x02, 0x03, 0x04:
		// 1. Check if it's an 8-byte Master Read Request
		if len(buf) >= 8 {
			candidate8 := buf[:8]
			rcvdCRC8 := uint16(candidate8[6]) | (uint16(candidate8[7]) << 8)
			if calculateCRC(candidate8[:6]) == rcvdCRC8 {
				return 8, candidate8
			}
		}

		// 2. Otherwise, parse as a Variable-Length Slave Response: [Slave][FC][ByteCount][Data...][CRC]
		if len(buf) >= 3 {
			byteCount := int(buf[2])
			// A valid Modbus RTU read response byte count must be reasonable (e.g., 1 to 252)
			if byteCount > 0 && byteCount <= 252 {
				expectedRespLen := 3 + byteCount + 2
				if len(buf) >= expectedRespLen {
					candidate := buf[:expectedRespLen]
					lowByte := candidate[len(candidate)-2]
					highByte := candidate[len(candidate)-1]
					receivedCRC := uint16(lowByte) | (uint16(highByte) << 8)
					if calculateCRC(candidate[:len(candidate)-2]) == receivedCRC {
						return expectedRespLen, candidate
					} else {
						// If it matched response structure by length/byteCount but failed CRC,
						// slide 1 byte to re-sync instead of waiting for a non-existent 8th byte.
						return 1, nil
					}
				} else {
					// We recognize it as a response, but we need more bytes to arrive
					return 0, nil
				}
			}
		}

		// If it's less than 8 bytes and not a valid response yet, wait for more data
		if len(buf) < 8 {
			return 0, nil
		}
		expectedLen = 8

	case 0x05, 0x06:
		expectedLen = 8
	case 0x0F, 0x10:
		if len(buf) >= 7 && funcCode == 0x10 {
			byteCount := int(buf[6])
			expectedLen = byteCount + 9
		} else {
			expectedLen = 8
		}

	default:
		if funcCode >= 0x80 {
			expectedLen = 5 // Exception response size
		} else {
			expectedLen = 8
		}
	}

	// If we still don't have enough bytes for the expected length, wait for more data
	if len(buf) < expectedLen {
		return 0, nil
	}

	candidate := buf[:expectedLen]
	lowByte := candidate[len(candidate)-2]
	highByte := candidate[len(candidate)-1]
	receivedCRC := uint16(lowByte) | (uint16(highByte) << 8)
	computedCRC := calculateCRC(candidate[:len(candidate)-2])

	if receivedCRC != computedCRC {
		// CRC failed: Slide window by 1 byte to re-sync stream alignment
		return 1, nil
	}

	return expectedLen, candidate
}

func calculateCRC(data []byte) uint16 {

	var crc uint16 = 0xFFFF
	for _, b := range data {
		crc ^= uint16(b)
		for range 8 {
			if (crc & 0x0001) != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}

func decodeDataPayload(funcCode byte, payload []byte, isResponse bool) []int16 {

	if len(payload) < 4 {
		return nil
	}

	var decoded []int16

	switch funcCode {
	case 0x03, 0x04:
		if isResponse && len(payload) >= 5 {
			byteCount := int(payload[2])
			expectedLen := 3 + byteCount + 2

			if len(payload) == expectedLen {
				regData := payload[3 : 3+byteCount]
				for i := 0; i+1 < len(regData); i += 2 {
					val := int16(binary.BigEndian.Uint16(regData[i : i+2]))
					decoded = append(decoded, val)
				}
			}
		}

	case 0x06:
		if len(payload) == 8 {
			val := int16(binary.BigEndian.Uint16(payload[4:6]))
			decoded = append(decoded, val)
		}

	case 0x10:
		if !isResponse && len(payload) > 9 {
			byteCount := int(payload[6])
			if len(payload) == 7+byteCount+2 {
				regData := payload[7 : 7+byteCount]
				for i := 0; i+1 < len(regData); i += 2 {
					val := int16(binary.BigEndian.Uint16(regData[i : i+2]))
					decoded = append(decoded, val)
				}
			}
		}
	}

	return decoded
}

var (
	lastRequestMutex sync.Mutex
	lastSlaveID      int
	lastFuncCode     byte
	lastRequestTime  time.Time
)

func (rtu *ModbusRTUSniffer) processRTUFrame(payload []byte) {
	if len(payload) < 4 {
		return
	}

	slaveID := int(payload[0])
	funcCode := payload[1]

	isResponse := false
	expectedLen := len(payload)

	lastRequestMutex.Lock()
	timeSinceLastReq := time.Since(lastRequestTime)

	switch funcCode {
	case 0x01, 0x02, 0x03, 0x04:
		// If it's exactly 8 bytes, it's a standard Modbus read request from Master -> Slave
		if len(payload) == 8 {
			isResponse = false
			expectedLen = 8
		} else {
			// Otherwise, it's a variable-length response from the Slave: [Slave][FC][ByteCount][Data...][CRC]
			isResponse = true
			if len(payload) >= 3 {
				byteCount := int(payload[2])
				expectedRespLen := 3 + byteCount + 2
				if len(payload) >= expectedRespLen {
					expectedLen = expectedRespLen
				}
			}
		}
	case 0x05, 0x06:
		expectedLen = 8
		if slaveID != 0 && lastSlaveID == slaveID && lastFuncCode == funcCode && timeSinceLastReq < 200*time.Millisecond {
			isResponse = true
			lastSlaveID = 0
			lastFuncCode = 0
		} else {
			isResponse = false
		}
	case 0x0F, 0x10:
		if len(payload) >= 7 && len(payload) == int(payload[6])+9 {
			isResponse = false
			expectedLen = int(payload[6]) + 9
		} else if len(payload) == 8 {
			isResponse = true
			expectedLen = 8
		}
	default:
		if funcCode >= 0x80 {
			isResponse = true
			expectedLen = 5
		}
	}

	if !isResponse {
		lastSlaveID = slaveID
		lastFuncCode = funcCode
		lastRequestTime = time.Now()
	}
	lastRequestMutex.Unlock()

	if expectedLen > 0 && len(payload) >= expectedLen {
		payload = payload[:expectedLen]
	}

	lowByte := payload[len(payload)-2]
	highByte := payload[len(payload)-1]

	receivedCRC := uint16(lowByte) | (uint16(highByte) << 8)
	computedCRC := calculateCRC(payload[:len(payload)-2])

	isValidCRC := (receivedCRC == computedCRC)

	typ := "REQ"
	source := "Master"
	destination := fmt.Sprintf("Slave %d", slaveID)

	if isResponse {
		typ = "RES"
		source = fmt.Sprintf("Slave %d", slaveID)
		destination = "Master"

		if !rtu.lastRequestTime.IsZero() {
			rtu.latence = float64(time.Since(lastRequestTime).Microseconds()) / 1000.0
		}

		rtu.counter.Response++
	} else {
		rtu.latence = 0
		rtu.lastRequestTime = time.Now()
		rtu.counter.Requests++
	}

	fcText := fcMap[funcCode]
	if fcText == "" {
		if funcCode >= 0x80 {
			fcText = fmt.Sprintf("Exception Error (0x%02X)", payload[2])
			rtu.counter.Exceptions++
		} else {
			fcText = "Unknown"
		}
	}

	now := time.Now()
	timestamp := now.Format("15:04:05") + fmt.Sprintf(".%03d", now.Nanosecond()/1000000)

	hexStrings := make([]string, len(payload))
	rawInts := make([]int, len(payload))
	for i, b := range payload {
		hexStrings[i] = fmt.Sprintf("0x%02X", b)
		rawInts[i] = int(b)
	}

	decodedPayload := decodeDataPayload(funcCode, payload, isResponse)

	frame := models.ModbusRTUFrame{
		Service:        RTU,
		Timestamp:      timestamp,
		Hex:            fmt.Sprintf("%X", payload),
		Type:           typ,
		Source:         source,
		Destination:    destination,
		BytesList:      hexStrings,
		RawInts:        rawInts,
		SlaveID:        slaveID,
		FC:             fmt.Sprintf("0x%02X (%s)", funcCode, fcText),
		CRC:            fmt.Sprintf("0x%02X%02X", lowByte, highByte),
		IsValidCRC:     isValidCRC,
		DecodedPayload: decodedPayload,
		Counter:        rtu.counter,
		Latency:        rtu.latence,
	}

	frameJSON, _ := json.Marshal(frame)
	rtu.webSocket.Broadcast(wsModels.TextMessage, frameJSON)
}

var simVariables struct {
	init     bool
	value    int
	response bool
}

var simValue uint16 = 0

func simulationRequest() []byte {
	// Slave ID (0x01), Function Code (0x03), Start Address (0x00, 0x00), Quantity: 10 registers (0x00, 0x0A)
	reqBase := []byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x0A}
	reqCRC := calculateCRC(reqBase)
	return append(reqBase, byte(reqCRC&0xFF), byte(reqCRC>>8))
}

func simulationResponse() []byte {
	respBase := []byte{
		0x01, // Slave ID
		0x03, // Function Code
		0x14, // Byte Count: 10 registers * 2 bytes = 20 bytes (0x14)
	}

	// Build values for 10 holding registers (indices 0 to 9)
	registers := make([]uint16, 10)
	for i := 0; i < 10; i++ {
		if i == 5 {
			// The 6th register (index 5) increments over time
			registers[i] = simValue
		} else {
			// Optional: assign standard or placeholder values to the other registers
			registers[i] = uint16(i * 10)
		}
	}

	// Append each register's high and low bytes (Modbus uses Big-Endian)
	for _, val := range registers {
		respBase = append(respBase, byte(val>>8), byte(val&0xFF))
	}

	respCRC := calculateCRC(respBase)
	frame := append(respBase, byte(respCRC&0xFF), byte(respCRC>>8))

	// Increment simValue for the next cycle (loops back at 100)
	simValue++
	if simValue > 100 {
		simValue = 0
	}

	return frame
}

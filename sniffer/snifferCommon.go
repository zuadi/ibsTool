package sniffer

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"ibsTool/models"

	wsModels "github.com/zuadi/webServer/models"
)

var fcMap = map[byte]string{
	0x01: "Read Coils", 0x02: "Read Discrete Inputs", 0x03: "Read Holding Registers",
	0x04: "Read Input Registers", 0x05: "Write Single Coil", 0x06: "Write Single Register",
	0x0F: "Write Multiple Coils", 0x10: "Write Multiple Registers", 0x17: "Read/Write Multiple Registers",
}

var (
	statsMutex      sync.Mutex
	byteFrequency   = make(map[byte]int)
	pendingRequests = make(map[string]time.Time)
	globalStats     struct {
		Requests   int
		Responses  int
		Exceptions int
		TopBytes   [][2]float64
	}
)

func processPayload(payload []byte, srcIP string, dstIP string, isDstPort502 bool, now time.Time, ws *wsModels.WSClient) {
	for _, b := range payload {
		byteFrequency[b]++
	}

	timestamp := now.Format("15:04:05") + fmt.Sprintf(".%03d", now.Nanosecond()/1000000)

	hexStrings := make([]string, len(payload))
	rawInts := make([]int, len(payload))
	for i, b := range payload {
		hexStrings[i] = fmt.Sprintf("0x%02X", b)
		rawInts[i] = int(b)
	}

	frame := models.ModbusFrame{
		Timestamp:   timestamp,
		Source:      srcIP,
		Destination: dstIP,
		Hex:         fmt.Sprintf("%X", payload),
		BytesList:   hexStrings,
		RawInts:     rawInts,
		Latency:     "0.00",
	}

	if len(payload) >= 7 {
		transID := uint16(payload[0])<<8 | uint16(payload[1])
		protocolID := uint16(payload[2])<<8 | uint16(payload[3])
		funcCode := payload[7]

		if protocolID == 0 {
			pduPayload := payload[7:]

			if isDstPort502 {
				globalStats.Requests++
				reqKey := fmt.Sprintf("%s:%s:%d", srcIP, dstIP, transID)
				pendingRequests[reqKey] = now

				frame.Type = "REQUEST"
				frame.FC = fmt.Sprintf("0x%02X (%s)", funcCode, fcMap[funcCode])
				if len(pduPayload) > 0 {
					frame.Integers = decodeIntegers(pduPayload)
				}
			} else {
				globalStats.Responses++
				reqKey := fmt.Sprintf("%s:%s:%d", dstIP, srcIP, transID)
				if reqTime, exists := pendingRequests[reqKey]; exists {
					frame.Latency = fmt.Sprintf("%.2f", float64(now.Sub(reqTime).Microseconds())/1000.0)
					delete(pendingRequests, reqKey)
				}

				if funcCode >= 0x80 {
					globalStats.Exceptions++
					errCode := byte(0x00)
					if len(payload) > 8 {
						errCode = payload[8]
					}
					frame.Type = "RESPONSE"
					frame.FC = fmt.Sprintf("0x%02X Exception", funcCode-0x80)
					frame.Exception = fmt.Sprintf("0x%02X Error", errCode)
				} else {
					frame.Type = "RESPONSE"
					frame.FC = fmt.Sprintf("0x%02X (%s)", funcCode, fcMap[funcCode])
					if len(pduPayload) > 0 {
						frame.Integers = decodeIntegers(pduPayload)
					}
				}
			}

			frameJSON, _ := json.Marshal(frame)
			ws.Broadcast(wsModels.TextMessage, frameJSON)
		}
	}

	globalStats.TopBytes = getTopBytes()
	statsJSON, _ := json.Marshal(map[string]any{
		"type":       "stats",
		"requests":   globalStats.Requests,
		"responses":  globalStats.Responses,
		"exceptions": globalStats.Exceptions,
		"top_bytes":  globalStats.TopBytes,
	})
	ws.Broadcast(wsModels.TextMessage, statsJSON)
}

func decodeIntegers(rawPduBytes []byte) []int {
	if len(rawPduBytes) <= 1 {
		return []int{}
	}
	dataBytes := rawPduBytes[1:]
	if len(dataBytes) > 1 && len(dataBytes)%2 != 0 {
		dataBytes = dataBytes[1:]
	}

	ints := make([]int, 0, len(dataBytes)/2)
	for i := 0; i < len(dataBytes)-1; i += 2 {
		val := int(dataBytes[i])<<8 | int(dataBytes[i+1])
		ints = append(ints, val)
	}
	return ints
}

func getTopBytes() [][2]float64 {
	type kv struct {
		Key   byte
		Value int
	}
	var ss []kv
	for k, v := range byteFrequency {
		ss = append(ss, kv{k, v})
	}
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})

	limit := 5
	if len(ss) < limit {
		limit = len(ss)
	}

	out := make([][2]float64, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, [2]float64{float64(ss[i].Key), float64(ss[i].Value)})
	}
	return out
}

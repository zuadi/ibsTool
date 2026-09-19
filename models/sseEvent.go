package models

import (
	"encoding/json"
	"fmt"
)

type SSEEvent struct {
	Event string `json:"-"`
	Data  any    `json:"-"`
}

func (e SSEEvent) Bytes() ([]byte, error) {
	dataBytes, err := json.Marshal(e.Data)
	if err != nil {
		return nil, err
	}

	if e.Event != "" {
		return fmt.Appendf(nil, "event: %s\ndata: %s\n\n", e.Event, dataBytes), nil
	}
	return fmt.Appendf(nil, "data: %s\n\n", dataBytes), nil
}

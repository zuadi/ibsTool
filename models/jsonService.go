package models

import "encoding/json"

func SetServiceAction(service, action string) []byte {
	payload, _ := json.Marshal(map[string]string{
		"service": service,
		"action":  action,
	})
	return payload
}

func SetServiceError(service string, err error) []byte {
	payload, _ := json.Marshal(map[string]string{
		"service": service,
		"error":   err.Error(),
	})
	return payload
}

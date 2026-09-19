package models

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	wsModels "github.com/zuadi/webServer/models"
)

func ReadJsonBody(in any, data any) error {
	var body io.ReadCloser
	switch i := in.(type) {
	case io.ReadCloser:
		body = i
	case wsModels.Context:
		body = i.GetRequest().Body
	case *http.Request:
		body = i.Body
	default:
		return errors.New("body type not supported")
	}

	content, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	return json.Unmarshal(content, data)
}

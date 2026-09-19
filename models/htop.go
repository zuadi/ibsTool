package models

import (
	"encoding/json"
	"io"
)

type Htop struct {
	Interval int `json:"interval"`
}

func GetInterval(body io.ReadCloser) (interval int, err error) {
	var b []byte
	b, err = io.ReadAll(body)
	if err != nil {
		return 0, err
	}

	var js Htop
	err = json.Unmarshal(b, &js)
	if err != nil {
		return 0, err
	}
	return js.Interval, nil
}

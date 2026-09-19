package tailscale

import "os"

type Config struct {
	Displayname string
	Server      string
	AuthKey     string
	StateDir    string
}

func LoadEnv() (*Config, error) {
	var err error
	displayname := os.Getenv("TS_DISPLAYNAME")
	server := os.Getenv("SERVER")
	authKey := os.Getenv("TS_AUTHKEY")
	stateDir := os.Getenv("TS_STATE_DIR")

	if stateDir == "" {
		stateDir = "./tsnet-state"
	}

	if displayname == "" {
		displayname, err = os.Hostname()
		if err != nil {
			return nil, err
		}
	}

	return &Config{
		Displayname: displayname,
		Server:      server,
		AuthKey:     authKey,
		StateDir:    stateDir,
	}, nil
}

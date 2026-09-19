package tailscale

import (
	"context"
	"fmt"
	"ibsTool/logging"
	"os"

	"tailscale.com/tsnet"
)

type Tailscale struct {
	config *Config
	server *tsnet.Server
	logger *logging.Logger
	cancel context.CancelFunc
}

func NewTailscaleClient(l *logging.Logger) (*Tailscale, error) {

	config, err := LoadEnv()
	if err != nil {
		return nil, err
	}

	if config.StateDir == "" {
		config.StateDir = "./tsnet-state"
	}

	if config.Displayname == "" {
		config.Displayname, err = os.Hostname()
		if err != nil {
			return nil, err
		}
	}

	if _, err := os.Stat(config.StateDir); err != nil {
		err := os.MkdirAll(config.StateDir, 0666)
		if err != nil {
			return nil, err
		}
	}

	return &Tailscale{config: config, logger: l}, nil
}

func (t *Tailscale) Connect() error {
	t.server = &tsnet.Server{
		// Unique hostname in your Tailnet
		Hostname: t.config.Displayname,

		// Directory where tsnet stores state/keys (ensure write access)
		Dir: t.config.StateDir,

		// Point directly to your Headscale Control Server URL
		ControlURL: t.config.Server,

		// Headscale Pre-Auth Key (generated via `headscale preauthkeys create`)
		AuthKey: t.config.AuthKey,

		//capture logs
		Logf: func(format string, args ...any) {
			msg := fmt.Sprintf(format, args...)
			if t.logger != nil {
				t.logger.BroadcastLog(msg)
			}
		},

		// Optional: Enable IP forwarding if this node will route actual traffic
		// System-level IP forwarding on the host machine must also be enabled.
	}

	// Start the embedded Tailscale server
	var ctx context.Context
	ctx, t.cancel = context.WithCancel(context.Background())
	status, err := t.server.Up(ctx)
	if err != nil {
		if t.logger != nil {
			t.logger.BroadcastLog(err)
		}
		return fmt.Errorf("Failed to start tsnet server: %v", err)
	}

	t.logger.BroadcastLog(fmt.Sprintf("Successfully connected to Headscale as %s (IP: %v)\n", t.server.Hostname, status.TailscaleIPs))

	// Keep running (or start your listener using s.Listen("tcp", ":80"))
	select {}
}

func (t *Tailscale) Disconnect() {
	t.logger.BroadcastLog("close server")
	if t.server == nil {
		t.logger.BroadcastLog("no tailscale server connected")

		return
	}
	if err := t.server.Close(); err != nil {
		t.logger.BroadcastLog(err)
	}
	t.logger.BroadcastLog("cancel context")
	t.cancel()
}

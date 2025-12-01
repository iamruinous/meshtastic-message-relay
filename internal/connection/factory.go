// Package connection provides Meshtastic connection implementations.
package connection

import (
	"fmt"

	"github.com/iamruinous/meshtastic-message-relay/internal/config"
)

// New creates a new Connection based on the configuration
func New(cfg config.ConnectionConfig) (Connection, error) {
	switch cfg.Type {
	case "serial":
		return NewSerial(cfg.Serial)
	case "tcp":
		return NewTCP(cfg.TCP)
	case "mqtt":
		return NewMQTT(cfg.MQTT)
	default:
		return nil, fmt.Errorf("unknown connection type: %s", cfg.Type)
	}
}

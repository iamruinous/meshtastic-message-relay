package output

import (
	"fmt"

	"github.com/iamruinous/meshtastic-message-relay/internal/config"
)

// New creates a new Output based on the configuration
func New(cfg config.OutputConfig) (Output, error) {
	switch cfg.Type {
	case "stdout":
		return NewStdout(cfg)
	case "file":
		return NewFile(cfg)
	case "apprise":
		return NewApprise(cfg)
	case "webhook":
		return NewWebhook(cfg)
	default:
		return nil, fmt.Errorf("unknown output type: %s", cfg.Type)
	}
}

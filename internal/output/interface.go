package output

import (
	"context"

	"github.com/iamruinous/meshtastic-message-relay/internal/message"
)

// Output defines the interface for message output destinations.
// Implementations include stdout, file, apprise, and webhook outputs.
type Output interface {
	// Send forwards a message to the output destination.
	// Returns an error if the message cannot be delivered.
	Send(ctx context.Context, msg *message.Packet) error

	// Close cleanly shuts down the output and releases any resources.
	Close() error

	// Name returns a unique identifier for this output.
	Name() string

	// Enabled returns true if this output is enabled and should receive messages.
	Enabled() bool
}

// Factory creates Output instances based on configuration.
type Factory interface {
	// Create creates a new Output instance based on the provided configuration.
	Create(cfg map[string]interface{}) (Output, error)
}

package connection

import (
	"context"

	"github.com/iamruinous/meshtastic-message-relay/internal/message"
)

// Connection defines the interface for Meshtastic node connections.
// Implementations include serial, TCP, and MQTT connections.
type Connection interface {
	// Connect establishes the connection to the Meshtastic node.
	// Returns an error if the connection cannot be established.
	Connect(ctx context.Context) error

	// Messages returns a channel that receives packets from the mesh network.
	// The channel is closed when the connection is closed.
	Messages() <-chan *message.Packet

	// Send transmits a packet to the mesh network.
	// Returns an error if the packet cannot be sent.
	Send(ctx context.Context, packet *message.Packet) error

	// Close cleanly shuts down the connection.
	// This should close the Messages channel.
	Close() error

	// Name returns a unique identifier for this connection.
	Name() string

	// IsConnected returns true if the connection is currently active.
	IsConnected() bool
}

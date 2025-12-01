package connection

import (
	"context"
	"fmt"

	"github.com/iamruinous/meshtastic-message-relay/internal/config"
	"github.com/iamruinous/meshtastic-message-relay/internal/message"
)

// TCP implements Connection for TCP connections
type TCP struct {
	config    config.TCPConfig
	messages  chan *message.Packet
	connected bool
}

// NewTCP creates a new TCP connection
func NewTCP(cfg config.TCPConfig) (*TCP, error) {
	return &TCP{
		config:   cfg,
		messages: make(chan *message.Packet, 100),
	}, nil
}

// Connect establishes the TCP connection
func (t *TCP) Connect(ctx context.Context) error {
	// TODO: Implement TCP connection to Meshtastic node
	return fmt.Errorf("TCP connection not yet implemented (host: %s, port: %d)", t.config.Host, t.config.Port)
}

// Messages returns the channel for receiving packets
func (t *TCP) Messages() <-chan *message.Packet {
	return t.messages
}

// Send transmits a packet over the TCP connection
func (t *TCP) Send(ctx context.Context, packet *message.Packet) error {
	if !t.connected {
		return fmt.Errorf("not connected")
	}
	// TODO: Implement sending
	return fmt.Errorf("send not yet implemented")
}

// Close closes the TCP connection
func (t *TCP) Close() error {
	t.connected = false
	close(t.messages)
	return nil
}

// Name returns the connection identifier
func (t *TCP) Name() string {
	return fmt.Sprintf("tcp:%s:%d", t.config.Host, t.config.Port)
}

// IsConnected returns the connection status
func (t *TCP) IsConnected() bool {
	return t.connected
}

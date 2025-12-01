package connection

import (
	"context"
	"fmt"

	"github.com/iamruinous/meshtastic-message-relay/internal/config"
	"github.com/iamruinous/meshtastic-message-relay/internal/message"
)

// Serial implements Connection for serial port connections
type Serial struct {
	config    config.SerialConfig
	messages  chan *message.Packet
	connected bool
}

// NewSerial creates a new serial connection
func NewSerial(cfg config.SerialConfig) (*Serial, error) {
	return &Serial{
		config:   cfg,
		messages: make(chan *message.Packet, 100),
	}, nil
}

// Connect establishes the serial connection
func (s *Serial) Connect(ctx context.Context) error {
	// TODO: Implement serial connection using go.bug.st/serial
	// For now, return a placeholder error
	return fmt.Errorf("serial connection not yet implemented (port: %s, baud: %d)", s.config.Port, s.config.Baud)
}

// Messages returns the channel for receiving packets
func (s *Serial) Messages() <-chan *message.Packet {
	return s.messages
}

// Send transmits a packet over the serial connection
func (s *Serial) Send(ctx context.Context, packet *message.Packet) error {
	if !s.connected {
		return fmt.Errorf("not connected")
	}
	// TODO: Implement sending
	return fmt.Errorf("send not yet implemented")
}

// Close closes the serial connection
func (s *Serial) Close() error {
	s.connected = false
	close(s.messages)
	return nil
}

// Name returns the connection identifier
func (s *Serial) Name() string {
	return fmt.Sprintf("serial:%s", s.config.Port)
}

// IsConnected returns the connection status
func (s *Serial) IsConnected() bool {
	return s.connected
}

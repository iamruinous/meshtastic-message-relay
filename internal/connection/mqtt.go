package connection

import (
	"context"
	"fmt"

	"github.com/iamruinous/meshtastic-message-relay/internal/config"
	"github.com/iamruinous/meshtastic-message-relay/internal/message"
)

// MQTT implements Connection for MQTT broker connections
type MQTT struct {
	config    config.MQTTConfig
	messages  chan *message.Packet
	connected bool
}

// NewMQTT creates a new MQTT connection
func NewMQTT(cfg config.MQTTConfig) (*MQTT, error) {
	return &MQTT{
		config:   cfg,
		messages: make(chan *message.Packet, 100),
	}, nil
}

// Connect establishes the MQTT connection
func (m *MQTT) Connect(ctx context.Context) error {
	// TODO: Implement MQTT connection using paho.mqtt.golang
	return fmt.Errorf("MQTT connection not yet implemented (broker: %s, topic: %s)", m.config.Broker, m.config.Topic)
}

// Messages returns the channel for receiving packets
func (m *MQTT) Messages() <-chan *message.Packet {
	return m.messages
}

// Send transmits a packet over MQTT
func (m *MQTT) Send(ctx context.Context, packet *message.Packet) error {
	if !m.connected {
		return fmt.Errorf("not connected")
	}
	// TODO: Implement sending
	return fmt.Errorf("send not yet implemented")
}

// Close closes the MQTT connection
func (m *MQTT) Close() error {
	m.connected = false
	close(m.messages)
	return nil
}

// Name returns the connection identifier
func (m *MQTT) Name() string {
	return fmt.Sprintf("mqtt:%s", m.config.Broker)
}

// IsConnected returns the connection status
func (m *MQTT) IsConnected() bool {
	return m.connected
}

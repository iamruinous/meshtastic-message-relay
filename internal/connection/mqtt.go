package connection

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"

	"github.com/iamruinous/meshtastic-message-relay/internal/config"
	"github.com/iamruinous/meshtastic-message-relay/internal/logging"
	"github.com/iamruinous/meshtastic-message-relay/internal/message"
	"github.com/iamruinous/meshtastic-message-relay/pkg/meshtastic"
)

// MQTT implements Connection for MQTT broker connections
type MQTT struct {
	config   config.MQTTConfig
	client   mqtt.Client
	messages chan *message.Packet
	nodeDB   map[uint32]*meshtastic.NodeInfo
	logger   *zap.Logger

	mu        sync.RWMutex
	connected bool
	stopCh    chan struct{}
}

// NewMQTT creates a new MQTT connection
func NewMQTT(cfg *config.MQTTConfig) (*MQTT, error) {
	return &MQTT{
		config:   *cfg,
		messages: make(chan *message.Packet, 100),
		nodeDB:   make(map[uint32]*meshtastic.NodeInfo),
		logger:   logging.With(zap.String("connection", "mqtt")),
		stopCh:   make(chan struct{}),
	}, nil
}

// Connect establishes the MQTT connection
func (m *MQTT) Connect(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.connected {
		return nil
	}

	m.logger.Info("Connecting to MQTT broker",
		zap.String("broker", m.config.Broker),
		zap.String("topic", m.config.Topic))

	// Generate client ID if not provided
	clientID := m.config.ClientID
	if clientID == "" {
		clientID = fmt.Sprintf("meshtastic-relay-%d", time.Now().UnixNano())
	}

	// Create MQTT client options
	opts := mqtt.NewClientOptions().
		AddBroker(m.config.Broker).
		SetClientID(clientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second).
		SetConnectionLostHandler(m.onConnectionLost).
		SetOnConnectHandler(m.onConnect)

	if m.config.Username != "" {
		opts.SetUsername(m.config.Username)
	}
	if m.config.Password != "" {
		opts.SetPassword(m.config.Password)
	}

	// Create and connect client
	client := mqtt.NewClient(opts)
	token := client.Connect()

	// Wait for connection with timeout
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("connection timeout")
	}
	if token.Error() != nil {
		return fmt.Errorf("failed to connect: %w", token.Error())
	}

	m.client = client
	m.connected = true
	m.stopCh = make(chan struct{})

	m.logger.Info("Connected to MQTT broker")
	return nil
}

// onConnect is called when the MQTT connection is established
func (m *MQTT) onConnect(client mqtt.Client) {
	m.logger.Info("MQTT connected, subscribing to topic", zap.String("topic", m.config.Topic))

	// Subscribe to the topic
	token := client.Subscribe(m.config.Topic, 1, m.messageHandler)
	if token.Wait() && token.Error() != nil {
		m.logger.Error("Failed to subscribe", zap.Error(token.Error()))
		return
	}

	m.mu.Lock()
	m.connected = true
	m.mu.Unlock()

	m.logger.Info("Subscribed to topic")
}

// onConnectionLost is called when the MQTT connection is lost
func (m *MQTT) onConnectionLost(_ mqtt.Client, err error) {
	m.logger.Warn("MQTT connection lost", zap.Error(err))

	m.mu.Lock()
	m.connected = false
	m.mu.Unlock()
}

// messageHandler processes incoming MQTT messages
func (m *MQTT) messageHandler(_ mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	payload := msg.Payload()

	m.logger.Debug("Received MQTT message",
		zap.String("topic", topic),
		zap.Int("size", len(payload)))

	// Parse the topic to extract message type
	// Meshtastic MQTT topics are typically: msh/region/channel/portnum/!nodeId
	packet := m.parseMessage(topic, payload)
	if packet == nil {
		return
	}

	// Send to channel (non-blocking)
	select {
	case m.messages <- packet:
	default:
		m.logger.Warn("Message channel full, dropping packet")
	}
}

// parseMessage parses an MQTT message into our packet format
func (m *MQTT) parseMessage(topic string, payload []byte) *message.Packet {
	// Try to parse as JSON first (some MQTT implementations use JSON)
	var jsonMsg struct {
		From     uint32      `json:"from"`
		To       uint32      `json:"to"`
		Channel  uint32      `json:"channel"`
		Type     string      `json:"type"`
		Payload  interface{} `json:"payload"`
		Sender   string      `json:"sender"`
		ID       uint32      `json:"id"`
		RxTime   int64       `json:"rxTime"`
		RxSnr    float32     `json:"rxSnr"`
		RxRssi   int32       `json:"rxRssi"`
		HopLimit uint32      `json:"hopLimit"`
	}

	if err := json.Unmarshal(payload, &jsonMsg); err == nil && jsonMsg.From != 0 {
		packet := &message.Packet{
			ID:         jsonMsg.ID,
			From:       jsonMsg.From,
			To:         jsonMsg.To,
			Channel:    jsonMsg.Channel,
			SNR:        jsonMsg.RxSnr,
			RSSI:       jsonMsg.RxRssi,
			HopLimit:   jsonMsg.HopLimit,
			ReceivedAt: time.Now(),
		}

		if jsonMsg.RxTime > 0 {
			packet.ReceivedAt = time.Unix(jsonMsg.RxTime, 0)
		}

		// Parse port number from type or topic
		packet.PortNum = m.parsePortNum(jsonMsg.Type, topic)

		// Set payload based on type
		switch packet.PortNum {
		case message.PortNumTextMessage:
			if text, ok := jsonMsg.Payload.(string); ok {
				packet.Payload = &message.TextMessage{Text: text}
			} else {
				packet.Payload = jsonMsg.Payload
			}
		default:
			packet.Payload = jsonMsg.Payload
		}

		return packet
	}

	// Try to parse as protobuf (binary format)
	// This handles the native Meshtastic MQTT format
	fromRadio, err := meshtastic.ParseFromRadio(payload)
	if err == nil && fromRadio.Packet != nil {
		meshPacket := fromRadio.ToPacket()
		if meshPacket != nil {
			return message.FromMeshtasticPacket(meshPacket)
		}
	}

	// If all else fails, treat as raw text message
	// Extract info from topic if possible
	parts := strings.Split(topic, "/")
	packet := &message.Packet{
		PortNum:    message.PortNumTextMessage,
		Payload:    &message.TextMessage{Text: string(payload)},
		ReceivedAt: time.Now(),
	}

	// Try to extract node ID from topic
	for _, part := range parts {
		if strings.HasPrefix(part, "!") && len(part) == 9 {
			// This looks like a node ID
			var nodeNum uint32
			_, _ = fmt.Sscanf(part, "!%08x", &nodeNum)
			if nodeNum != 0 {
				packet.From = nodeNum
			}
		}
	}

	return packet
}

// parsePortNum extracts the port number from type string or topic
func (m *MQTT) parsePortNum(typeStr, topic string) message.PortNum {
	typeStr = strings.ToUpper(typeStr)

	portMap := map[string]message.PortNum{
		"TEXT_MESSAGE_APP": message.PortNumTextMessage,
		"TEXT":             message.PortNumTextMessage,
		"POSITION_APP":     message.PortNumPosition,
		"POSITION":         message.PortNumPosition,
		"NODEINFO_APP":     message.PortNumNodeInfo,
		"NODEINFO":         message.PortNumNodeInfo,
		"TELEMETRY_APP":    message.PortNumTelemetry,
		"TELEMETRY":        message.PortNumTelemetry,
		"ROUTING_APP":      message.PortNumRouting,
		"ROUTING":          message.PortNumRouting,
		"TRACEROUTE_APP":   message.PortNumTraceroute,
		"TRACEROUTE":       message.PortNumTraceroute,
		"NEIGHBORINFO_APP": message.PortNumNeighborInfo,
		"NEIGHBORINFO":     message.PortNumNeighborInfo,
	}

	if portNum, ok := portMap[typeStr]; ok {
		return portNum
	}

	// Try to find type in topic
	topicUpper := strings.ToUpper(topic)
	for key, portNum := range portMap {
		if strings.Contains(topicUpper, key) {
			return portNum
		}
	}

	return message.PortNumUnknown
}

// Messages returns the channel for receiving packets
func (m *MQTT) Messages() <-chan *message.Packet {
	return m.messages
}

// Send transmits a packet over MQTT
func (m *MQTT) Send(_ context.Context, _ *message.Packet) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.connected {
		return fmt.Errorf("not connected")
	}

	// TODO: Implement sending
	return fmt.Errorf("send not yet implemented")
}

// Close closes the MQTT connection
func (m *MQTT) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return nil
	}

	m.logger.Info("Closing MQTT connection")

	// Signal stop
	close(m.stopCh)
	m.connected = false

	// Disconnect client
	if m.client != nil && m.client.IsConnected() {
		m.client.Disconnect(1000) // Wait up to 1 second
	}

	// Close message channel
	close(m.messages)

	return nil
}

// Name returns the connection identifier
func (m *MQTT) Name() string {
	return fmt.Sprintf("mqtt:%s", m.config.Broker)
}

// IsConnected returns the connection status
func (m *MQTT) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected && m.client != nil && m.client.IsConnected()
}

// GetNodeInfo returns information about a specific node
func (m *MQTT) GetNodeInfo(nodeNum uint32) *meshtastic.NodeInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.nodeDB[nodeNum]
}

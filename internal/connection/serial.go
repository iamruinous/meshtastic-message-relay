package connection

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.bug.st/serial"
	"go.uber.org/zap"

	"github.com/iamruinous/meshtastic-message-relay/internal/config"
	"github.com/iamruinous/meshtastic-message-relay/internal/logging"
	"github.com/iamruinous/meshtastic-message-relay/internal/message"
	"github.com/iamruinous/meshtastic-message-relay/pkg/meshtastic"
)

// Serial implements Connection for serial port connections
type Serial struct {
	config    config.SerialConfig
	port      serial.Port
	framer    *meshtastic.StreamFramer
	messages  chan *message.Packet
	nodeDB    map[uint32]*meshtastic.NodeInfo
	myInfo    *meshtastic.MyNodeInfo
	logger    *zap.Logger

	mu        sync.RWMutex
	connected bool
	stopCh    chan struct{}
}

// NewSerial creates a new serial connection
func NewSerial(cfg config.SerialConfig) (*Serial, error) {
	return &Serial{
		config:   cfg,
		messages: make(chan *message.Packet, 100),
		nodeDB:   make(map[uint32]*meshtastic.NodeInfo),
		logger:   logging.With(zap.String("connection", "serial")),
		stopCh:   make(chan struct{}),
	}, nil
}

// Connect establishes the serial connection
func (s *Serial) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connected {
		return nil
	}

	s.logger.Info("Connecting to serial port",
		zap.String("port", s.config.Port),
		zap.Int("baud", s.config.Baud))

	// Open serial port
	mode := &serial.Mode{
		BaudRate: s.config.Baud,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(s.config.Port, mode)
	if err != nil {
		return fmt.Errorf("failed to open serial port: %w", err)
	}

	// Set read timeout
	if err := port.SetReadTimeout(100 * time.Millisecond); err != nil {
		port.Close()
		return fmt.Errorf("failed to set read timeout: %w", err)
	}

	s.port = port
	s.framer = meshtastic.NewStreamFramer(port, port)
	s.connected = true
	s.stopCh = make(chan struct{})

	// Start the read loop
	go s.readLoop(ctx)

	// Request initial config
	go s.requestConfig()

	s.logger.Info("Connected to serial port")
	return nil
}

// Messages returns the channel for receiving packets
func (s *Serial) Messages() <-chan *message.Packet {
	return s.messages
}

// Send transmits a packet over the serial connection
func (s *Serial) Send(ctx context.Context, packet *message.Packet) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.connected {
		return fmt.Errorf("not connected")
	}

	// TODO: Implement sending - requires encoding ToRadio protobuf
	return fmt.Errorf("send not yet implemented")
}

// Close closes the serial connection
func (s *Serial) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected {
		return nil
	}

	s.logger.Info("Closing serial connection")

	// Signal stop
	close(s.stopCh)
	s.connected = false

	// Close port
	if s.port != nil {
		if err := s.port.Close(); err != nil {
			s.logger.Error("Error closing serial port", zap.Error(err))
		}
		s.port = nil
	}

	// Close message channel
	close(s.messages)

	return nil
}

// Name returns the connection identifier
func (s *Serial) Name() string {
	return fmt.Sprintf("serial:%s", s.config.Port)
}

// IsConnected returns the connection status
func (s *Serial) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected
}

// readLoop continuously reads packets from the serial port
func (s *Serial) readLoop(ctx context.Context) {
	s.logger.Debug("Starting read loop")

	for {
		select {
		case <-ctx.Done():
			s.logger.Debug("Read loop stopped: context cancelled")
			return
		case <-s.stopCh:
			s.logger.Debug("Read loop stopped: stop signal")
			return
		default:
			s.readPacket()
		}
	}
}

func (s *Serial) readPacket() {
	// Read a framed packet
	data, err := s.framer.ReadPacket()
	if err != nil {
		// Timeout is expected, don't log it
		if err.Error() != "EOF" {
			s.logger.Debug("Error reading packet", zap.Error(err))
		}
		return
	}

	// Parse the FromRadio message
	fromRadio, err := meshtastic.ParseFromRadio(data)
	if err != nil {
		s.logger.Debug("Error parsing FromRadio", zap.Error(err))
		return
	}

	s.handleFromRadio(fromRadio)
}

func (s *Serial) handleFromRadio(fr *meshtastic.FromRadio) {
	// Handle different message types
	if fr.MyInfo != nil {
		s.mu.Lock()
		s.myInfo = fr.MyInfo
		s.mu.Unlock()
		s.logger.Info("Received MyInfo",
			zap.Uint32("node_num", fr.MyInfo.MyNodeNum))
	}

	if fr.NodeInfo != nil {
		s.mu.Lock()
		s.nodeDB[fr.NodeInfo.Num] = fr.NodeInfo
		s.mu.Unlock()

		userName := ""
		if fr.NodeInfo.User != nil {
			userName = fr.NodeInfo.User.LongName
		}
		s.logger.Debug("Received NodeInfo",
			zap.Uint32("num", fr.NodeInfo.Num),
			zap.String("name", userName))
	}

	if fr.ConfigCompleteID != 0 {
		s.logger.Info("Config complete", zap.Uint32("id", fr.ConfigCompleteID))
	}

	if fr.Packet != nil {
		// Convert to internal packet format
		meshPacket := fr.ToPacket()
		if meshPacket == nil {
			return
		}

		// Attach node info if available
		s.mu.RLock()
		if nodeInfo, ok := s.nodeDB[meshPacket.From]; ok {
			meshPacket.FromNode = nodeInfo
		}
		s.mu.RUnlock()

		// Convert to our message format
		packet := message.FromMeshtasticPacket(meshPacket)
		if packet == nil {
			return
		}

		s.logger.Debug("Received packet",
			zap.Uint32("from", packet.From),
			zap.Uint32("to", packet.To),
			zap.String("port", packet.PortNum.String()))

		// Send to channel (non-blocking)
		select {
		case s.messages <- packet:
		default:
			s.logger.Warn("Message channel full, dropping packet")
		}
	}
}

// requestConfig sends a request for initial configuration
func (s *Serial) requestConfig() {
	// Wait a moment for the connection to stabilize
	time.Sleep(500 * time.Millisecond)

	s.mu.RLock()
	connected := s.connected
	s.mu.RUnlock()

	if !connected {
		return
	}

	// Send a WantConfig request
	// ToRadio { want_config_id: 1 }
	// Field 3, varint wire type, value 1
	// Tag: (3 << 3) | 0 = 24 = 0x18
	wantConfig := []byte{0x18, 0x01}

	s.logger.Debug("Requesting initial configuration")
	if err := s.framer.WritePacket(wantConfig); err != nil {
		s.logger.Error("Failed to request config", zap.Error(err))
	}
}

// GetNodeInfo returns information about a specific node
func (s *Serial) GetNodeInfo(nodeNum uint32) *meshtastic.NodeInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nodeDB[nodeNum]
}

// GetMyInfo returns information about this node
func (s *Serial) GetMyInfo() *meshtastic.MyNodeInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.myInfo
}

package connection

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/iamruinous/meshtastic-message-relay/internal/config"
	"github.com/iamruinous/meshtastic-message-relay/internal/logging"
	"github.com/iamruinous/meshtastic-message-relay/internal/message"
	"github.com/iamruinous/meshtastic-message-relay/pkg/meshtastic"
)

// TCP implements Connection for TCP connections
type TCP struct {
	config   config.TCPConfig
	conn     net.Conn
	framer   *meshtastic.StreamFramer
	messages chan *message.Packet
	nodeDB   map[uint32]*meshtastic.NodeInfo
	myInfo   *meshtastic.MyNodeInfo
	logger   *zap.Logger

	mu        sync.RWMutex
	connected bool
	stopCh    chan struct{}
}

// NewTCP creates a new TCP connection
func NewTCP(cfg config.TCPConfig) (*TCP, error) {
	return &TCP{
		config:   cfg,
		messages: make(chan *message.Packet, 100),
		nodeDB:   make(map[uint32]*meshtastic.NodeInfo),
		logger:   logging.With(zap.String("connection", "tcp")),
		stopCh:   make(chan struct{}),
	}, nil
}

// Connect establishes the TCP connection
func (t *TCP) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected {
		return nil
	}

	addr := fmt.Sprintf("%s:%d", t.config.Host, t.config.Port)
	t.logger.Info("Connecting to TCP endpoint", zap.String("address", addr))

	// Dial with timeout
	dialer := net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	// Set read deadline for non-blocking reads
	_ = conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

	t.conn = conn
	t.framer = meshtastic.NewStreamFramer(conn, conn)
	t.connected = true
	t.stopCh = make(chan struct{})

	// Start the read loop
	go t.readLoop(ctx)

	// Request initial config
	go t.requestConfig()

	t.logger.Info("Connected to TCP endpoint")
	return nil
}

// Messages returns the channel for receiving packets
func (t *TCP) Messages() <-chan *message.Packet {
	return t.messages
}

// Send transmits a packet over the TCP connection
func (t *TCP) Send(_ context.Context, _ *message.Packet) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.connected {
		return fmt.Errorf("not connected")
	}

	// TODO: Implement sending - requires encoding ToRadio protobuf
	return fmt.Errorf("send not yet implemented")
}

// Close closes the TCP connection
func (t *TCP) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected {
		return nil
	}

	t.logger.Info("Closing TCP connection")

	// Signal stop
	close(t.stopCh)
	t.connected = false

	// Close connection
	if t.conn != nil {
		if err := t.conn.Close(); err != nil {
			t.logger.Error("Error closing TCP connection", zap.Error(err))
		}
		t.conn = nil
	}

	// Close message channel
	close(t.messages)

	return nil
}

// Name returns the connection identifier
func (t *TCP) Name() string {
	return fmt.Sprintf("tcp:%s:%d", t.config.Host, t.config.Port)
}

// IsConnected returns the connection status
func (t *TCP) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.connected
}

// readLoop continuously reads packets from the TCP connection
func (t *TCP) readLoop(ctx context.Context) {
	t.logger.Debug("Starting read loop")

	for {
		select {
		case <-ctx.Done():
			t.logger.Debug("Read loop stopped: context cancelled")
			return
		case <-t.stopCh:
			t.logger.Debug("Read loop stopped: stop signal")
			return
		default:
			t.readPacket()
		}
	}
}

func (t *TCP) readPacket() {
	// Reset read deadline
	t.mu.RLock()
	if t.conn != nil {
		_ = t.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	}
	t.mu.RUnlock()

	// Read a framed packet
	data, err := t.framer.ReadPacket()
	if err != nil {
		// Timeout is expected, don't log it
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return
		}
		if err.Error() != "EOF" {
			t.logger.Debug("Error reading packet", zap.Error(err))
		}
		return
	}

	// Parse the FromRadio message
	fromRadio, err := meshtastic.ParseFromRadio(data)
	if err != nil {
		t.logger.Debug("Error parsing FromRadio", zap.Error(err))
		return
	}

	t.handleFromRadio(fromRadio)
}

func (t *TCP) handleFromRadio(fr *meshtastic.FromRadio) {
	// Handle different message types
	if fr.MyInfo != nil {
		t.mu.Lock()
		t.myInfo = fr.MyInfo
		t.mu.Unlock()
		t.logger.Info("Received MyInfo",
			zap.Uint32("node_num", fr.MyInfo.MyNodeNum))
	}

	if fr.NodeInfo != nil {
		t.mu.Lock()
		t.nodeDB[fr.NodeInfo.Num] = fr.NodeInfo
		t.mu.Unlock()

		userName := ""
		if fr.NodeInfo.User != nil {
			userName = fr.NodeInfo.User.LongName
		}
		t.logger.Debug("Received NodeInfo",
			zap.Uint32("num", fr.NodeInfo.Num),
			zap.String("name", userName))
	}

	if fr.ConfigCompleteID != 0 {
		t.logger.Info("Config complete", zap.Uint32("id", fr.ConfigCompleteID))
	}

	if fr.Packet != nil {
		// Convert to internal packet format
		meshPacket := fr.ToPacket()
		if meshPacket == nil {
			return
		}

		// Attach node info if available
		t.mu.RLock()
		if nodeInfo, ok := t.nodeDB[meshPacket.From]; ok {
			meshPacket.FromNode = nodeInfo
		}
		t.mu.RUnlock()

		// Convert to our message format
		packet := message.FromMeshtasticPacket(meshPacket)
		if packet == nil {
			return
		}

		t.logger.Debug("Received packet",
			zap.Uint32("from", packet.From),
			zap.Uint32("to", packet.To),
			zap.String("port", packet.PortNum.String()))

		// Send to channel (non-blocking)
		select {
		case t.messages <- packet:
		default:
			t.logger.Warn("Message channel full, dropping packet")
		}
	}
}

// requestConfig sends a request for initial configuration
func (t *TCP) requestConfig() {
	// Wait a moment for the connection to stabilize
	time.Sleep(500 * time.Millisecond)

	t.mu.RLock()
	connected := t.connected
	t.mu.RUnlock()

	if !connected {
		return
	}

	// Send a WantConfig request
	wantConfig := []byte{0x18, 0x01}

	t.logger.Debug("Requesting initial configuration")
	if err := t.framer.WritePacket(wantConfig); err != nil {
		t.logger.Error("Failed to request config", zap.Error(err))
	}
}

// GetNodeInfo returns information about a specific node
func (t *TCP) GetNodeInfo(nodeNum uint32) *meshtastic.NodeInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.nodeDB[nodeNum]
}

// GetMyInfo returns information about this node
func (t *TCP) GetMyInfo() *meshtastic.MyNodeInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.myInfo
}

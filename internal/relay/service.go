package relay

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/iamruinous/meshtastic-message-relay/internal/config"
	"github.com/iamruinous/meshtastic-message-relay/internal/connection"
	"github.com/iamruinous/meshtastic-message-relay/internal/logging"
	"github.com/iamruinous/meshtastic-message-relay/internal/message"
	"github.com/iamruinous/meshtastic-message-relay/internal/output"
)

// Service orchestrates the message relay between connections and outputs
type Service struct {
	config     *config.Config
	connection connection.Connection
	outputs    []output.Output
	logger     *zap.Logger

	mu       sync.RWMutex
	running  bool
	stats    Stats
	messages chan *message.Packet
}

// Stats holds runtime statistics for the relay service
type Stats struct {
	MessagesReceived uint64
	MessagesSent     uint64
	MessagesFiltered uint64
	Errors           uint64
}

// New creates a new relay service with the given configuration
func New(cfg *config.Config) (*Service, error) {
	logger := logging.With(zap.String("component", "relay"))

	return &Service{
		config:   cfg,
		logger:   logger,
		messages: make(chan *message.Packet, 100),
	}, nil
}

// Start initializes the connection and outputs, then begins relaying messages
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("service is already running")
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Info("Starting relay service")

	// Initialize outputs
	if err := s.initOutputs(); err != nil {
		return fmt.Errorf("failed to initialize outputs: %w", err)
	}

	// Initialize connection
	if err := s.initConnection(); err != nil {
		s.closeOutputs()
		return fmt.Errorf("failed to initialize connection: %w", err)
	}

	// Connect to the Meshtastic node
	if err := s.connection.Connect(ctx); err != nil {
		s.closeOutputs()
		return fmt.Errorf("failed to connect: %w", err)
	}

	s.logger.Info("Relay service started",
		zap.String("connection", s.connection.Name()),
		zap.Int("outputs", len(s.outputs)))

	// Start the message relay loop
	go s.relayLoop(ctx)

	return nil
}

// Stop gracefully shuts down the relay service
func (s *Service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.logger.Info("Stopping relay service")
	s.running = false

	// Close connection
	if s.connection != nil {
		if err := s.connection.Close(); err != nil {
			s.logger.Error("Error closing connection", zap.Error(err))
		}
	}

	// Close outputs
	s.closeOutputs()

	s.logger.Info("Relay service stopped")
	return nil
}

// IsRunning returns true if the service is currently running
func (s *Service) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetStats returns the current runtime statistics
func (s *Service) GetStats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

// GetConnection returns the current connection (may be nil)
func (s *Service) GetConnection() connection.Connection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connection
}

// GetOutputs returns the configured outputs
func (s *Service) GetOutputs() []output.Output {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.outputs
}

func (s *Service) initConnection() error {
	var err error
	s.connection, err = connection.New(s.config.Connection)
	return err
}

func (s *Service) initOutputs() error {
	s.outputs = make([]output.Output, 0)

	for _, outCfg := range s.config.Outputs {
		if !outCfg.Enabled {
			continue
		}

		out, err := output.New(outCfg)
		if err != nil {
			return fmt.Errorf("failed to create output %s: %w", outCfg.Type, err)
		}
		s.outputs = append(s.outputs, out)
		s.logger.Debug("Initialized output", zap.String("type", outCfg.Type), zap.String("name", out.Name()))
	}

	if len(s.outputs) == 0 {
		return fmt.Errorf("no outputs enabled")
	}

	return nil
}

func (s *Service) closeOutputs() {
	for _, out := range s.outputs {
		if err := out.Close(); err != nil {
			s.logger.Error("Error closing output", zap.String("output", out.Name()), zap.Error(err))
		}
	}
}

func (s *Service) relayLoop(ctx context.Context) {
	msgChan := s.connection.Messages()

	for {
		select {
		case <-ctx.Done():
			s.logger.Debug("Relay loop stopped: context cancelled")
			return

		case msg, ok := <-msgChan:
			if !ok {
				s.logger.Debug("Relay loop stopped: message channel closed")
				return
			}

			s.mu.Lock()
			s.stats.MessagesReceived++
			s.mu.Unlock()

			// Apply filters
			if !s.shouldRelay(msg) {
				s.mu.Lock()
				s.stats.MessagesFiltered++
				s.mu.Unlock()
				continue
			}

			// Send to all outputs
			s.sendToOutputs(ctx, msg)
		}
	}
}

func (s *Service) shouldRelay(msg *message.Packet) bool {
	filters := s.config.Filters

	// Filter by message type
	if len(filters.MessageTypes) > 0 {
		msgType := msg.PortNum.String()
		found := false
		for _, t := range filters.MessageTypes {
			if t == msgType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by node ID
	if len(filters.NodeIDs) > 0 {
		found := false
		for _, id := range filters.NodeIDs {
			if id == msg.From {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by channel
	if len(filters.Channels) > 0 {
		found := false
		for _, ch := range filters.Channels {
			if ch == msg.Channel {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func (s *Service) sendToOutputs(ctx context.Context, msg *message.Packet) {
	for _, out := range s.outputs {
		if err := out.Send(ctx, msg); err != nil {
			s.logger.Error("Failed to send message to output",
				zap.String("output", out.Name()),
				zap.Error(err))
			s.mu.Lock()
			s.stats.Errors++
			s.mu.Unlock()
		} else {
			s.mu.Lock()
			s.stats.MessagesSent++
			s.mu.Unlock()
		}
	}
}

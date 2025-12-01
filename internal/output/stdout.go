package output

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/iamruinous/meshtastic-message-relay/internal/config"
	"github.com/iamruinous/meshtastic-message-relay/internal/message"
)

// Stdout outputs messages to standard output
type Stdout struct {
	format  string
	enabled bool
}

// NewStdout creates a new stdout output
func NewStdout(cfg config.OutputConfig) (*Stdout, error) {
	format := "json"
	if f, ok := cfg.Options["format"].(string); ok {
		format = f
	}

	return &Stdout{
		format:  format,
		enabled: cfg.Enabled,
	}, nil
}

// Send outputs a message to stdout
func (s *Stdout) Send(ctx context.Context, msg *message.Packet) error {
	if s.format == "json" {
		return s.sendJSON(msg)
	}
	return s.sendText(msg)
}

func (s *Stdout) sendJSON(msg *message.Packet) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	fmt.Fprintln(os.Stdout, string(data))
	return nil
}

func (s *Stdout) sendText(msg *message.Packet) error {
	timestamp := msg.ReceivedAt.Format(time.RFC3339)
	fromNode := fmt.Sprintf("!%08x", msg.From)
	if msg.FromNode != nil && msg.FromNode.User != nil {
		fromNode = msg.FromNode.User.ShortName
	}

	var payload string
	switch p := msg.Payload.(type) {
	case *message.TextMessage:
		payload = p.Text
	case string:
		payload = p
	default:
		payload = fmt.Sprintf("%v", msg.Payload)
	}

	fmt.Fprintf(os.Stdout, "[%s] %s (%s): %s\n",
		timestamp,
		fromNode,
		msg.PortNum.String(),
		payload,
	)
	return nil
}

// Close closes the stdout output (no-op)
func (s *Stdout) Close() error {
	return nil
}

// Name returns the output identifier
func (s *Stdout) Name() string {
	return "stdout"
}

// Enabled returns whether this output is enabled
func (s *Stdout) Enabled() bool {
	return s.enabled
}

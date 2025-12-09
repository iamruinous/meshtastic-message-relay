package output

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/iamruinous/meshtastic-message-relay/internal/config"
	"github.com/iamruinous/meshtastic-message-relay/internal/message"
)

// File outputs messages to a file
type File struct {
	path       string
	format     string
	enabled    bool
	rotate     bool
	maxSizeMB  int
	maxBackups int

	mu   sync.Mutex
	file *os.File
}

// NewFile creates a new file output
func NewFile(cfg config.OutputConfig) (*File, error) {
	path := "/var/log/meshtastic/messages.log"
	if p, ok := cfg.Options["path"].(string); ok {
		path = p
	}

	format := "json"
	if f, ok := cfg.Options["format"].(string); ok {
		format = f
	}

	rotate := true
	if r, ok := cfg.Options["rotate"].(bool); ok {
		rotate = r
	}

	maxSizeMB := 100
	switch m := cfg.Options["max_size_mb"].(type) {
	case int:
		maxSizeMB = m
	case float64:
		maxSizeMB = int(m)
	}

	maxBackups := 5
	switch m := cfg.Options["max_backups"].(type) {
	case int:
		maxBackups = m
	case float64:
		maxBackups = int(m)
	}

	f := &File{
		path:       path,
		format:     format,
		enabled:    cfg.Enabled,
		rotate:     rotate,
		maxSizeMB:  maxSizeMB,
		maxBackups: maxBackups,
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open file for appending
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	f.file = file

	return f, nil
}

// Send writes a message to the file
func (f *File) Send(_ context.Context, msg *message.Packet) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.rotate {
		if err := f.checkRotation(); err != nil {
			return err
		}
	}

	var line string
	if f.format == "json" {
		data, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal message: %w", err)
		}
		line = string(data) + "\n"
	} else {
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

		line = fmt.Sprintf("[%s] %s (%s): %s\n", timestamp, fromNode, msg.PortNum.String(), payload)
	}

	_, err := f.file.WriteString(line)
	return err
}

func (f *File) checkRotation() error {
	info, err := f.file.Stat()
	if err != nil {
		return err
	}

	maxBytes := int64(f.maxSizeMB) * 1024 * 1024
	if info.Size() < maxBytes {
		return nil
	}

	// Close current file
	_ = f.file.Close()

	// Rotate existing backups
	for i := f.maxBackups - 1; i >= 1; i-- {
		oldPath := fmt.Sprintf("%s.%d", f.path, i)
		newPath := fmt.Sprintf("%s.%d", f.path, i+1)
		_ = os.Rename(oldPath, newPath)
	}

	// Rename current to .1
	_ = os.Rename(f.path, f.path+".1")

	// Open new file
	file, err := os.OpenFile(f.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	f.file = file

	return nil
}

// Close closes the file
func (f *File) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.file != nil {
		return f.file.Close()
	}
	return nil
}

// Name returns the output identifier
func (f *File) Name() string {
	return fmt.Sprintf("file:%s", f.path)
}

// Enabled returns whether this output is enabled
func (f *File) Enabled() bool {
	return f.enabled
}

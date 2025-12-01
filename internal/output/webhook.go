package output

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/iamruinous/meshtastic-message-relay/internal/config"
	"github.com/iamruinous/meshtastic-message-relay/internal/message"
)

// Webhook outputs messages to a generic HTTP webhook
type Webhook struct {
	url     string
	method  string
	timeout time.Duration
	headers map[string]string
	enabled bool
	client  *http.Client
}

// NewWebhook creates a new webhook output
func NewWebhook(cfg config.OutputConfig) (*Webhook, error) {
	url := ""
	if u, ok := cfg.Options["url"].(string); ok {
		url = u
	}
	if url == "" {
		return nil, fmt.Errorf("webhook url is required")
	}

	method := "POST"
	if m, ok := cfg.Options["method"].(string); ok {
		method = m
	}

	timeout := 30 * time.Second
	if t, ok := cfg.Options["timeout"].(string); ok {
		if d, err := time.ParseDuration(t); err == nil {
			timeout = d
		}
	}

	headers := make(map[string]string)
	if h, ok := cfg.Options["headers"].(map[string]interface{}); ok {
		for k, v := range h {
			if s, ok := v.(string); ok {
				headers[k] = s
			}
		}
	}

	return &Webhook{
		url:     url,
		method:  method,
		timeout: timeout,
		headers: headers,
		enabled: cfg.Enabled,
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Send sends a message to the webhook
func (w *Webhook) Send(ctx context.Context, msg *message.Packet) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, w.method, w.url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set default content type if not specified
	if _, ok := w.headers["Content-Type"]; !ok {
		req.Header.Set("Content-Type", "application/json")
	}

	for k, v := range w.headers {
		req.Header.Set(k, v)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send to webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// Close closes the webhook output
func (w *Webhook) Close() error {
	return nil
}

// Name returns the output identifier
func (w *Webhook) Name() string {
	return fmt.Sprintf("webhook:%s", w.url)
}

// Enabled returns whether this output is enabled
func (w *Webhook) Enabled() bool {
	return w.enabled
}

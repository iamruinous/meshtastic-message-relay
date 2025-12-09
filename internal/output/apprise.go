// Package output provides message output implementations.
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

// AppriseChannelConfig holds per-channel settings for Apprise
type AppriseChannelConfig struct {
	Tag     string
	Enabled *bool // nil means inherit from parent
}

// Apprise outputs messages to an Apprise notification service
type Apprise struct {
	url            string
	tag            string
	timeout        time.Duration
	headers        map[string]string
	enabled        bool
	client         *http.Client
	channelConfigs map[uint32]AppriseChannelConfig
}

// ApprisePayload is the JSON payload sent to Apprise
type ApprisePayload struct {
	Body  string `json:"body"`
	Title string `json:"title,omitempty"`
	Type  string `json:"type,omitempty"`
	Tag   string `json:"tag,omitempty"`
}

// NewApprise creates a new Apprise output
func NewApprise(cfg config.OutputConfig) (*Apprise, error) {
	url := ""
	if u, ok := cfg.Options["url"].(string); ok {
		url = u
	}
	if url == "" {
		return nil, fmt.Errorf("apprise url is required")
	}

	tag := "meshtastic"
	if t, ok := cfg.Options["tag"].(string); ok {
		tag = t
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

	// Parse per-channel configurations
	channelConfigs := make(map[uint32]AppriseChannelConfig)
	if channels, ok := cfg.Options["channels"].(map[string]interface{}); ok {
		for chStr, chCfg := range channels {
			var channelNum uint32
			if _, err := fmt.Sscanf(chStr, "%d", &channelNum); err != nil {
				continue // skip invalid channel numbers
			}

			if chMap, ok := chCfg.(map[string]interface{}); ok {
				cc := AppriseChannelConfig{}
				if t, ok := chMap["tag"].(string); ok {
					cc.Tag = t
				}
				if e, ok := chMap["enabled"].(bool); ok {
					cc.Enabled = &e
				}
				channelConfigs[channelNum] = cc
			}
		}
	}

	return &Apprise{
		url:            url,
		tag:            tag,
		timeout:        timeout,
		headers:        headers,
		enabled:        cfg.Enabled,
		channelConfigs: channelConfigs,
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Send sends a message to Apprise
func (a *Apprise) Send(ctx context.Context, msg *message.Packet) error {
	// Check for per-channel enabled override
	if chCfg, ok := a.channelConfigs[msg.Channel]; ok && chCfg.Enabled != nil {
		if !*chCfg.Enabled {
			return nil // Channel is explicitly disabled
		}
	}

	title := a.formatTitle(msg)
	body := a.formatBody(msg)

	// Use per-channel tag if configured, otherwise fall back to default
	tag := a.tag
	if chCfg, ok := a.channelConfigs[msg.Channel]; ok && chCfg.Tag != "" {
		tag = chCfg.Tag
	}

	payload := ApprisePayload{
		Body:  body,
		Title: title,
		Type:  "info",
		Tag:   tag,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal apprise payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range a.headers {
		req.Header.Set(k, v)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send to apprise: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("apprise returned status %d", resp.StatusCode)
	}

	return nil
}

func (a *Apprise) formatTitle(msg *message.Packet) string {
	fromNode := fmt.Sprintf("!%08x", msg.From)
	if msg.FromNode != nil && msg.FromNode.User != nil {
		fromNode = msg.FromNode.User.LongName
		if fromNode == "" {
			fromNode = msg.FromNode.User.ShortName
		}
	}
	return fmt.Sprintf("Meshtastic: %s", fromNode)
}

func (a *Apprise) formatBody(msg *message.Packet) string {
	switch p := msg.Payload.(type) {
	case *message.TextMessage:
		return p.Text
	case string:
		return p
	default:
		return fmt.Sprintf("[%s] %v", msg.PortNum.String(), msg.Payload)
	}
}

// Close closes the Apprise output
func (a *Apprise) Close() error {
	return nil
}

// Name returns the output identifier
func (a *Apprise) Name() string {
	return fmt.Sprintf("apprise:%s", a.url)
}

// Enabled returns whether this output is enabled
func (a *Apprise) Enabled() bool {
	return a.enabled
}

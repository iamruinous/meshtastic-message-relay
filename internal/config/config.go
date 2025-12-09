// Package config provides configuration types and loading for the relay service.
package config

import "time"

// Config represents the complete application configuration.
type Config struct {
	Connection ConnectionConfig `mapstructure:"connection"`
	Outputs    []OutputConfig   `mapstructure:"outputs"`
	Filters    FilterConfig     `mapstructure:"filters"`
	Logging    LoggingConfig    `mapstructure:"logging"`
}

// ConnectionConfig defines how to connect to the Meshtastic node.
type ConnectionConfig struct {
	Type   string       `mapstructure:"type"` // serial, tcp, mqtt
	Serial SerialConfig `mapstructure:"serial"`
	TCP    TCPConfig    `mapstructure:"tcp"`
	MQTT   MQTTConfig   `mapstructure:"mqtt"`
}

// SerialConfig defines serial port connection settings.
type SerialConfig struct {
	Port string `mapstructure:"port"`
	Baud int    `mapstructure:"baud"`
}

// TCPConfig defines TCP connection settings.
type TCPConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// MQTTConfig defines MQTT connection settings.
type MQTTConfig struct {
	Broker   string `mapstructure:"broker"`
	Topic    string `mapstructure:"topic"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	ClientID string `mapstructure:"client_id"`
}

// OutputConfig defines a single output destination.
type OutputConfig struct {
	Type    string                 `mapstructure:"type"` // stdout, file, apprise, webhook
	Enabled bool                   `mapstructure:"enabled"`
	Options map[string]interface{} `mapstructure:",remain"`
}

// StdoutOutputConfig defines stdout output settings.
type StdoutOutputConfig struct {
	Format string `mapstructure:"format"` // json, text
}

// FileOutputConfig defines file output settings.
type FileOutputConfig struct {
	Path       string `mapstructure:"path"`
	Format     string `mapstructure:"format"` // json, text
	Rotate     bool   `mapstructure:"rotate"`
	MaxSizeMB  int    `mapstructure:"max_size_mb"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAgeDays int    `mapstructure:"max_age_days"`
}

// AppriseOutputConfig defines Apprise output settings.
type AppriseOutputConfig struct {
	URL      string                          `mapstructure:"url"`
	Tag      string                          `mapstructure:"tag"`
	Timeout  time.Duration                   `mapstructure:"timeout"`
	Headers  map[string]string               `mapstructure:"headers"`
	Channels map[uint32]AppriseChannelConfig `mapstructure:"channels"`
}

// AppriseChannelConfig defines per-channel Apprise settings.
type AppriseChannelConfig struct {
	Tag     string `mapstructure:"tag"`
	Enabled *bool  `mapstructure:"enabled"` // nil means inherit from parent
}

// WebhookOutputConfig defines webhook output settings.
type WebhookOutputConfig struct {
	URL     string            `mapstructure:"url"`
	Method  string            `mapstructure:"method"`
	Headers map[string]string `mapstructure:"headers"`
	Timeout time.Duration     `mapstructure:"timeout"`
}

// FilterConfig defines message filtering rules.
type FilterConfig struct {
	MessageTypes []string `mapstructure:"message_types"`
	NodeIDs      []uint32 `mapstructure:"node_ids"`
	Channels     []uint32 `mapstructure:"channels"`
}

// LoggingConfig defines logging settings.
type LoggingConfig struct {
	Level  string `mapstructure:"level"`  // debug, info, warn, error
	Format string `mapstructure:"format"` // json, text
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Connection: ConnectionConfig{
			Type: "serial",
			Serial: SerialConfig{
				Port: "/dev/ttyUSB0",
				Baud: 115200,
			},
			TCP: TCPConfig{
				Host: "localhost",
				Port: 4403,
			},
			MQTT: MQTTConfig{
				Broker: "tcp://localhost:1883",
				Topic:  "meshtastic/#",
			},
		},
		Outputs: []OutputConfig{
			{
				Type:    "stdout",
				Enabled: true,
				Options: map[string]interface{}{
					"format": "json",
				},
			},
		},
		Filters: FilterConfig{
			MessageTypes: []string{},
			NodeIDs:      []uint32{},
			Channels:     []uint32{},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}
}

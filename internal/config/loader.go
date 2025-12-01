package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Load reads the configuration from viper and returns a Config struct
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Connection settings
	cfg.Connection.Type = viper.GetString("connection.type")

	// Serial settings
	cfg.Connection.Serial.Port = viper.GetString("connection.serial.port")
	cfg.Connection.Serial.Baud = viper.GetInt("connection.serial.baud")
	if cfg.Connection.Serial.Baud == 0 {
		cfg.Connection.Serial.Baud = 115200
	}

	// TCP settings
	cfg.Connection.TCP.Host = viper.GetString("connection.tcp.host")
	cfg.Connection.TCP.Port = viper.GetInt("connection.tcp.port")
	if cfg.Connection.TCP.Port == 0 {
		cfg.Connection.TCP.Port = 4403
	}

	// MQTT settings
	cfg.Connection.MQTT.Broker = viper.GetString("connection.mqtt.broker")
	cfg.Connection.MQTT.Topic = viper.GetString("connection.mqtt.topic")
	cfg.Connection.MQTT.Username = viper.GetString("connection.mqtt.username")
	cfg.Connection.MQTT.Password = viper.GetString("connection.mqtt.password")
	cfg.Connection.MQTT.ClientID = viper.GetString("connection.mqtt.client_id")

	// Load outputs
	outputsRaw := viper.Get("outputs")
	if outputsRaw != nil {
		if outputs, ok := outputsRaw.([]interface{}); ok {
			cfg.Outputs = make([]OutputConfig, 0, len(outputs))
			for _, out := range outputs {
				if outMap, ok := out.(map[string]interface{}); ok {
					outputCfg := OutputConfig{
						Type:    getString(outMap, "type"),
						Enabled: getBool(outMap, "enabled"),
						Options: outMap,
					}
					cfg.Outputs = append(cfg.Outputs, outputCfg)
				}
			}
		}
	}

	// Filters
	cfg.Filters.MessageTypes = viper.GetStringSlice("filters.message_types")
	cfg.Filters.NodeIDs = toUint32Slice(viper.Get("filters.node_ids"))
	cfg.Filters.Channels = toUint32Slice(viper.Get("filters.channels"))

	// Logging
	cfg.Logging.Level = viper.GetString("logging.level")
	cfg.Logging.Format = viper.GetString("logging.format")
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "text"
	}

	return cfg, nil
}

// Validate checks the configuration for errors
func (c *Config) Validate() error {
	// Validate connection type
	switch c.Connection.Type {
	case "serial", "tcp", "mqtt":
		// Valid
	case "":
		return fmt.Errorf("connection.type is required")
	default:
		return fmt.Errorf("invalid connection.type: %s (must be serial, tcp, or mqtt)", c.Connection.Type)
	}

	// Validate connection-specific settings
	switch c.Connection.Type {
	case "serial":
		if c.Connection.Serial.Port == "" {
			return fmt.Errorf("connection.serial.port is required for serial connection")
		}
	case "tcp":
		if c.Connection.TCP.Host == "" {
			return fmt.Errorf("connection.tcp.host is required for tcp connection")
		}
	case "mqtt":
		if c.Connection.MQTT.Broker == "" {
			return fmt.Errorf("connection.mqtt.broker is required for mqtt connection")
		}
	}

	// Validate outputs
	if len(c.Outputs) == 0 {
		return fmt.Errorf("at least one output must be configured")
	}

	enabledOutputs := 0
	for i, out := range c.Outputs {
		if out.Enabled {
			enabledOutputs++
		}
		if out.Type == "" {
			return fmt.Errorf("outputs[%d].type is required", i)
		}
		switch out.Type {
		case "stdout", "file", "apprise", "webhook":
			// Valid
		default:
			return fmt.Errorf("outputs[%d].type is invalid: %s", i, out.Type)
		}
	}

	if enabledOutputs == 0 {
		return fmt.Errorf("at least one output must be enabled")
	}

	return nil
}

// Helper functions

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getDuration(m map[string]interface{}, key string, defaultVal time.Duration) time.Duration {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case string:
			if d, err := time.ParseDuration(val); err == nil {
				return d
			}
		case int:
			return time.Duration(val) * time.Second
		case int64:
			return time.Duration(val) * time.Second
		case float64:
			return time.Duration(val) * time.Second
		}
	}
	return defaultVal
}

func toUint32Slice(v interface{}) []uint32 {
	if v == nil {
		return nil
	}
	switch slice := v.(type) {
	case []interface{}:
		result := make([]uint32, 0, len(slice))
		for _, item := range slice {
			switch n := item.(type) {
			case int:
				result = append(result, uint32(n))
			case int64:
				result = append(result, uint32(n))
			case float64:
				result = append(result, uint32(n))
			}
		}
		return result
	case []int:
		result := make([]uint32, len(slice))
		for i, n := range slice {
			result[i] = uint32(n)
		}
		return result
	}
	return nil
}

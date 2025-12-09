# Meshtastic Message Relay

[![Build Status](https://github.com/iamruinous/meshtastic-message-relay/actions/workflows/build.yml/badge.svg)](https://github.com/iamruinous/meshtastic-message-relay/actions/workflows/build.yml)
[![Release](https://img.shields.io/github/v/release/iamruinous/meshtastic-message-relay)](https://github.com/iamruinous/meshtastic-message-relay/releases)
[![Docker](https://img.shields.io/docker/v/iamruinous/meshtastic-message-relay?label=docker)](https://hub.docker.com/r/iamruinous/meshtastic-message-relay)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A flexible, extensible message relay service for [Meshtastic](https://meshtastic.org/) nodes. Listen for messages from your mesh network and forward them to various notification services, files, or custom endpoints.

## Features

- **Multiple Connection Methods**
  - Serial (USB-connected nodes)
  - TCP (network-connected nodes)
  - MQTT (broker-based communication)

- **Flexible Output Destinations**
  - **stdout** - Console output for debugging or piping
  - **File** - Write messages to log files with rotation support
  - **Apprise** - Send to 80+ notification services via [Apprise](https://github.com/caronc/apprise)
  - **Webhook** - Forward to any HTTP endpoint
  - *Easily extensible for custom outputs*

- **Powerful Filtering**
  - Filter by message type (text, position, telemetry, etc.)
  - Filter by node ID
  - Filter by channel

- **Production Ready**
  - Graceful startup and shutdown
  - Structured logging (JSON or text)
  - Docker and Kubernetes ready
  - Prometheus metrics (optional)

## Quick Start

### Using Docker

```bash
docker run -d \
  --name meshtastic-relay \
  -v /path/to/config.yaml:/etc/meshtastic-relay/config.yaml \
  --device=/dev/ttyUSB0 \
  iamruinous/meshtastic-message-relay
```

### Using Docker Compose

```yaml
version: '3.8'
services:
  meshtastic-relay:
    image: iamruinous/meshtastic-message-relay:latest
    volumes:
      - ./config.yaml:/etc/meshtastic-relay/config.yaml
      - ./logs:/var/log/meshtastic
    devices:
      - /dev/ttyUSB0:/dev/ttyUSB0
    restart: unless-stopped
```

### Using Binary

Download the latest release from the [releases page](https://github.com/iamruinous/meshtastic-message-relay/releases).

```bash
# Run with a configuration file
./meshtastic-relay --config config.yaml

# Or use environment variables and flags
./meshtastic-relay --connection.type=serial --connection.serial.port=/dev/ttyUSB0
```

### Building from Source

```bash
# Clone the repository
git clone https://github.com/iamruinous/meshtastic-message-relay.git
cd meshtastic-message-relay

# Build
go build -o meshtastic-relay ./cmd/relay

# Or use make
make build
```

### Building with Nix

This project includes a Nix flake for reproducible builds:

```bash
# Build the package
nix build

# Run directly without installing
nix run

# Enter development shell
nix develop

# Or with direnv (automatic shell activation)
direnv allow
```

## Configuration

Create a `config.yaml` file:

```yaml
# Connection to your Meshtastic node
connection:
  type: serial  # Options: serial, tcp, mqtt

  serial:
    port: /dev/ttyUSB0
    baud: 115200

  # tcp:
  #   host: 192.168.1.100
  #   port: 4403

  # mqtt:
  #   broker: tcp://localhost:1883
  #   topic: meshtastic/#
  #   username: ""
  #   password: ""

# Output destinations - enable one or more
outputs:
  # Console output
  - type: stdout
    enabled: true
    format: json  # Options: json, text

  # File logging
  - type: file
    enabled: false
    path: /var/log/meshtastic/messages.log
    format: json
    rotate: true
    max_size_mb: 100
    max_backups: 5

  # Apprise notifications
  - type: apprise
    enabled: false
    url: http://apprise:8000/notify
    tag: meshtastic
    # Supports all Apprise notification services:
    # Discord, Slack, Telegram, Email, Pushover, etc.

  # Custom webhook
  - type: webhook
    enabled: false
    url: https://your-service.com/webhook
    method: POST
    headers:
      Content-Type: application/json
      Authorization: "Bearer ${WEBHOOK_TOKEN}"

# Message filtering (optional)
filters:
  # Only relay specific message types
  message_types:
    - TEXT_MESSAGE_APP
    - POSITION_APP
    - TELEMETRY_APP
    - NODEINFO_APP

  # Only relay from specific nodes (empty = all)
  node_ids: []

  # Only relay from specific channels (empty = all)
  channels: []

# Logging configuration
logging:
  level: info   # Options: debug, info, warn, error
  format: json  # Options: json, text
```

### Environment Variables

All configuration options can be set via environment variables using the prefix `MESH_RELAY_`:

```bash
export MESH_RELAY_CONNECTION_TYPE=serial
export MESH_RELAY_CONNECTION_SERIAL_PORT=/dev/ttyUSB0
export MESH_RELAY_LOGGING_LEVEL=debug
```

## Apprise Integration

[Apprise](https://github.com/caronc/apprise) provides a unified interface to send notifications to 80+ services. Run Apprise as a sidecar:

```yaml
version: '3.8'
services:
  meshtastic-relay:
    image: iamruinous/meshtastic-message-relay:latest
    volumes:
      - ./config.yaml:/etc/meshtastic-relay/config.yaml
    devices:
      - /dev/ttyUSB0:/dev/ttyUSB0
    depends_on:
      - apprise

  apprise:
    image: caronc/apprise:latest
    ports:
      - "8000:8000"
    volumes:
      - ./apprise.yaml:/config/apprise.yaml
    environment:
      - APPRISE_CONFIG_FILE=/config/apprise.yaml
```

Example Apprise configuration (`apprise.yaml`):

```yaml
urls:
  - discord://webhook_id/webhook_token
  - slack://token_a/token_b/token_c
  - tgram://bot_token/chat_id
```

## Message Types

The relay can handle various Meshtastic message types:

| Type | Description |
|------|-------------|
| `TEXT_MESSAGE_APP` | Text messages sent between nodes |
| `POSITION_APP` | GPS position updates |
| `TELEMETRY_APP` | Device telemetry (battery, sensors) |
| `NODEINFO_APP` | Node information updates |
| `ROUTING_APP` | Routing information |
| `WAYPOINT_APP` | Waypoint data |

## Architecture

```
┌──────────────────┐     ┌─────────────────┐     ┌──────────────────┐
│   Meshtastic     │     │                 │     │     Outputs      │
│      Node        │────▶│  Message Relay  │────▶│  (Apprise, File, │
│ (Serial/TCP/MQTT)│     │                 │     │   Webhook, etc.) │
└──────────────────┘     └─────────────────┘     └──────────────────┘
                                │
                                ▼
                         ┌─────────────┐
                         │   Filters   │
                         │ & Routing   │
                         └─────────────┘
```

The relay uses a plugin-based architecture, making it easy to add new connection types or output destinations. See [CLAUDE.md](CLAUDE.md) for development details.

## Development

### Prerequisites

- Go 1.21 or later
- Make (optional)

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Run linter
make lint
```

### Adding Custom Outputs

The relay is designed to be extensible. Implement the `Output` interface to add new destinations:

```go
type Output interface {
    Send(ctx context.Context, msg *message.Packet) error
    Close() error
    Name() string
}
```

See existing implementations in `internal/output/` for examples.

## Roadmap

### Completed

- [x] Core relay service implementation
- [x] Serial connection support
- [x] TCP connection support
- [x] MQTT connection support
- [x] stdout output
- [x] File output with rotation
- [x] Apprise integration
- [x] Generic webhook output
- [x] CLI framework with Cobra
- [x] Interactive TUI with Bubbletea
- [x] Configuration management with Viper
- [x] Structured logging with Zap
- [x] GitHub Actions CI/CD (build, release, docker)
- [x] Dockerfile and docker-compose
- [x] Nix flake for reproducible builds
- [x] Device simulator for testing (PTY-based)
- [x] Meshtastic protocol framing/parsing

### In Progress

- [ ] Message filtering by type/node/channel
- [ ] Docker image publishing to GHCR

### Planned

- [ ] Prometheus metrics endpoint
- [ ] Web UI for status monitoring
- [ ] Position/telemetry specific outputs
- [ ] Node database persistence
- [ ] Message acknowledgment support
- [ ] Rate limiting for outputs
- [ ] Retry logic with exponential backoff
- [ ] Health check endpoint
- [ ] Graceful degradation when outputs fail
- [ ] Integration tests with real devices
- [ ] Plugin system for custom outputs

## Inspiration

This project is inspired by:

- [meshtastic-matrix-relay](https://github.com/geoffwhittington/meshtastic-matrix-relay) - Matrix integration for Meshtastic
- [meshtastic-bridge](https://github.com/pdxlocations/meshtastic-bridge) - Bridge between Meshtastic networks

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- [Open an issue](https://github.com/iamruinous/meshtastic-message-relay/issues) for bug reports or feature requests
- [Meshtastic Discord](https://discord.gg/meshtastic) for general Meshtastic discussion

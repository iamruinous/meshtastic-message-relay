# CLAUDE.md - Development Guide for Meshtastic Message Relay

## Project Overview

Meshtastic Message Relay is a Go service that listens for messages from Meshtastic nodes and forwards them to configurable endpoints. It supports multiple connection methods (serial, TCP, MQTT) and multiple output destinations (Apprise, filesystem, stdout, webhooks).

## Architecture

### Core Design Principles

1. **Plugin-based Architecture**: Both connections (sources) and outputs (sinks) are implemented as plugins with a common interface
2. **Message Pipeline**: Messages flow through: Connection → Parser → Router → Output(s)
3. **Configuration-driven**: YAML configuration for flexible deployment
4. **Graceful Lifecycle**: Clean startup/shutdown with proper resource cleanup

### Directory Structure

```
meshtastic-message-relay/
├── cmd/
│   └── relay/              # Main application entry point
│       └── main.go
├── internal/
│   ├── config/             # Configuration loading and validation
│   ├── connection/         # Connection interface and implementations
│   │   ├── interface.go    # Connection interface definition
│   │   ├── serial.go       # Serial port connection
│   │   ├── tcp.go          # TCP connection
│   │   └── mqtt.go         # MQTT connection
│   ├── message/            # Message types and parsing
│   │   ├── types.go        # Message struct definitions
│   │   └── parser.go       # Protobuf parsing
│   ├── output/             # Output interface and implementations
│   │   ├── interface.go    # Output interface definition
│   │   ├── stdout.go       # Console output
│   │   ├── file.go         # File system output
│   │   ├── apprise.go      # Apprise notification service
│   │   └── webhook.go      # Generic HTTP webhook
│   ├── router/             # Message routing logic
│   │   └── router.go       # Routes messages to outputs
│   └── relay/              # Core relay service
│       └── service.go      # Main service orchestration
├── pkg/
│   └── meshtastic/         # Meshtastic protocol helpers (if needed)
├── configs/
│   └── example.yaml        # Example configuration
├── scripts/
│   ├── build.sh            # Build script
│   └── release.sh          # Release automation
├── .github/
│   └── workflows/
│       ├── build.yml       # CI build workflow
│       ├── release.yml     # Release workflow
│       └── docker.yml      # Docker build workflow
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Key Interfaces

### Connection Interface

```go
type Connection interface {
    // Connect establishes the connection to the Meshtastic node
    Connect(ctx context.Context) error

    // Messages returns a channel of received messages
    Messages() <-chan *message.Packet

    // Close cleanly shuts down the connection
    Close() error

    // Name returns the connection identifier
    Name() string
}
```

### Output Interface

```go
type Output interface {
    // Send forwards a message to the output destination
    Send(ctx context.Context, msg *message.Packet) error

    // Close cleanly shuts down the output
    Close() error

    // Name returns the output identifier
    Name() string
}
```

## Build Commands

```bash
# Development
go run ./cmd/relay

# Build binary
go build -o meshtastic-relay ./cmd/relay

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Build Docker image
docker build -t meshtastic-relay .

# Run with Docker Compose
docker-compose up
```

## Configuration

Configuration is loaded from YAML files. Example:

```yaml
# Connection to Meshtastic node
connection:
  type: serial  # serial, tcp, or mqtt
  serial:
    port: /dev/ttyUSB0
    baud: 115200
  tcp:
    host: 192.168.1.100
    port: 4403
  mqtt:
    broker: tcp://localhost:1883
    topic: meshtastic/#

# Output destinations (multiple can be enabled)
outputs:
  - type: stdout
    enabled: true
    format: json  # json, text

  - type: file
    enabled: true
    path: /var/log/meshtastic/messages.log
    format: json
    rotate: true
    max_size_mb: 100

  - type: apprise
    enabled: true
    url: http://localhost:8000/notify
    tag: meshtastic

  - type: webhook
    enabled: true
    url: https://example.com/webhook
    method: POST
    headers:
      Authorization: Bearer ${WEBHOOK_TOKEN}

# Filtering (optional)
filters:
  message_types:
    - TEXT_MESSAGE_APP
    - POSITION_APP
  node_ids: []  # Empty means all nodes
  channels: []  # Empty means all channels

# Logging
logging:
  level: info  # debug, info, warn, error
  format: json # json, text
```

## Dependencies

Key dependencies to use:

- `github.com/meshtastic/go` - Meshtastic protobuf definitions and helpers (if available, otherwise generate from proto files)
- `github.com/spf13/viper` - Configuration management
- `github.com/spf13/cobra` - CLI framework
- `go.bug.st/serial` - Serial port communication
- `github.com/eclipse/paho.mqtt.golang` - MQTT client
- `go.uber.org/zap` - Structured logging
- `github.com/prometheus/client_golang` - Metrics (optional)

## Testing Strategy

1. **Unit Tests**: Test individual components in isolation
2. **Integration Tests**: Test connection and output implementations with mocks
3. **End-to-End Tests**: Test full message flow with simulated inputs

## Error Handling

- Use structured errors with context
- Implement retry logic for transient failures
- Log errors with appropriate severity levels
- Graceful degradation when outputs fail

## Performance Considerations

- Use buffered channels for message passing
- Implement connection pooling for HTTP outputs
- Consider rate limiting for high-volume scenarios
- Profile and optimize hot paths

## Common Development Tasks

### Adding a New Output Type

1. Create a new file in `internal/output/`
2. Implement the `Output` interface
3. Register the output type in the output factory
4. Add configuration schema support
5. Write unit tests
6. Update documentation

### Adding a New Connection Type

1. Create a new file in `internal/connection/`
2. Implement the `Connection` interface
3. Register the connection type in the connection factory
4. Add configuration schema support
5. Write unit tests
6. Update documentation

## Release Process

1. Update version in relevant files
2. Update CHANGELOG.md
3. Create a git tag: `git tag v1.0.0`
4. Push tag: `git push origin v1.0.0`
5. GitHub Actions will automatically:
   - Build binaries for all platforms
   - Create GitHub release with binaries
   - Build and push Docker image

## Debugging Tips

- Enable debug logging: `--log-level=debug`
- Use `--dry-run` to test configuration without sending
- Check connection status with `--status` command
- Inspect message flow with stdout output enabled

## Related Projects

- [meshtastic-matrix-relay](https://github.com/geoffwhittington/meshtastic-matrix-relay) - Matrix integration
- [meshtastic-bridge](https://github.com/pdxlocations/meshtastic-bridge) - Bridge between mesh networks
- [meshtastic/go](https://github.com/meshtastic/go) - Official Go library

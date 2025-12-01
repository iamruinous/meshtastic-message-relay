// Package simulator provides a Meshtastic device simulator for testing.
package simulator

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/iamruinous/meshtastic-message-relay/pkg/meshtastic"
)

// DeviceConfig holds configuration for the simulated device
type DeviceConfig struct {
	// NodeNum is this device's node number
	NodeNum uint32
	// LongName is the device's long name
	LongName string
	// ShortName is the device's short name (4 chars)
	ShortName string
	// HWModel is the hardware model ID
	HWModel uint32
	// Latitude in degrees
	Latitude float64
	// Longitude in degrees
	Longitude float64
	// Altitude in meters
	Altitude int32
	// SimulatedNodes are other nodes in the mesh
	SimulatedNodes []SimulatedNode
	// MessageInterval is how often to send simulated messages (0 = manual only)
	MessageInterval time.Duration
	// Verbose enables verbose logging
	Verbose bool
}

// SimulatedNode represents another node in the simulated mesh
type SimulatedNode struct {
	NodeNum   uint32
	LongName  string
	ShortName string
	HWModel   uint32
	Latitude  float64
	Longitude float64
	Altitude  int32
}

// DefaultConfig returns a default device configuration
func DefaultConfig() DeviceConfig {
	return DeviceConfig{
		NodeNum:   0x12345678,
		LongName:  "Simulated Node",
		ShortName: "SIM1",
		HWModel:   9, // TBEAM
		Latitude:  37.7749,
		Longitude: -122.4194,
		Altitude:  10,
		SimulatedNodes: []SimulatedNode{
			{
				NodeNum:   0xAABBCCDD,
				LongName:  "Remote Node 1",
				ShortName: "REM1",
				HWModel:   9,
				Latitude:  37.7849,
				Longitude: -122.4094,
				Altitude:  20,
			},
			{
				NodeNum:   0x11223344,
				LongName:  "Remote Node 2",
				ShortName: "REM2",
				HWModel:   14, // HELTEC
				Latitude:  37.7649,
				Longitude: -122.4294,
				Altitude:  15,
			},
		},
		MessageInterval: 30 * time.Second,
	}
}

// Device simulates a Meshtastic device
type Device struct {
	config DeviceConfig
	pty    *PTY
	framer *meshtastic.StreamFramer
	logger func(format string, args ...interface{})

	mu         sync.RWMutex
	running    bool
	stopCh     chan struct{}
	packetID   uint32
	configSent bool
}

// New creates a new simulated device
func New(config DeviceConfig) *Device {
	logger := func(_ string, _ ...interface{}) {}
	if config.Verbose {
		logger = func(format string, args ...interface{}) {
			fmt.Printf("[SIM] "+format+"\n", args...)
		}
	}

	return &Device{
		config:   config,
		logger:   logger,
		stopCh:   make(chan struct{}),
		packetID: uint32(rand.Intn(10000)),
	}
}

// Start starts the simulated device and returns the path to connect to
func (d *Device) Start(ctx context.Context) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.running {
		return d.pty.SlavePath, nil
	}

	// Create PTY
	pty, err := OpenPTY()
	if err != nil {
		return "", fmt.Errorf("failed to create PTY: %w", err)
	}

	d.pty = pty
	d.framer = meshtastic.NewStreamFramer(pty.Master, pty.Master)
	d.running = true
	d.stopCh = make(chan struct{})
	d.configSent = false

	// Start the read loop
	go d.readLoop(ctx)

	// Start the message generator if interval is set
	if d.config.MessageInterval > 0 {
		go d.messageLoop(ctx)
	}

	d.logger("Device started on %s", pty.SlavePath)
	return pty.SlavePath, nil
}

// Stop stops the simulated device
func (d *Device) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.running {
		return nil
	}

	d.logger("Stopping device")
	close(d.stopCh)
	d.running = false

	if d.pty != nil {
		_ = d.pty.Close()
		d.pty = nil
	}

	return nil
}

// Config returns the device configuration
func (d *Device) Config() DeviceConfig {
	return d.config
}

// GetPath returns the path to the slave PTY device
func (d *Device) GetPath() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.pty != nil {
		return d.pty.SlavePath
	}
	return ""
}

// SendTextMessage sends a simulated text message from a remote node
func (d *Device) SendTextMessage(fromNode uint32, text string) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if !d.running {
		return fmt.Errorf("device not running")
	}

	d.packetID++
	packet := d.createTextMessagePacket(fromNode, text)
	return d.sendFromRadio(packet, nil, nil, 0)
}

// SendPosition sends a simulated position update from a remote node
func (d *Device) SendPosition(fromNode uint32, lat, lon float64, alt int32) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if !d.running {
		return fmt.Errorf("device not running")
	}

	d.packetID++
	packet := d.createPositionPacket(fromNode, lat, lon, alt)
	return d.sendFromRadio(packet, nil, nil, 0)
}

func (d *Device) readLoop(ctx context.Context) {
	d.logger("Starting read loop")

	for {
		// Check if we should stop
		select {
		case <-ctx.Done():
			return
		case <-d.stopCh:
			return
		default:
		}

		// Use short deadline to allow checking stop conditions
		// but still allow blocking reads to work
		_ = d.pty.Master.SetReadDeadline(time.Now().Add(500 * time.Millisecond))

		data, err := d.framer.ReadPacket()
		if err != nil {
			// Expected errors - just retry
			if isExpectedError(err) {
				continue
			}
			// Log unexpected errors
			if d.config.Verbose && err.Error() != "EOF" {
				d.logger("ReadPacket error: %v", err)
			}
			continue
		}

		d.logger("Received packet: %d bytes: %v", len(data), data)
		d.handleToRadio(data)
	}
}

// isExpectedError checks if an error is expected (timeout or EIO before slave is opened)
func isExpectedError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// i/o timeout - normal when waiting for data
	// resource temporarily unavailable - EAGAIN on non-blocking read
	// input/output error - EIO when slave is not yet opened or was closed
	return strings.Contains(errStr, "i/o timeout") ||
		strings.Contains(errStr, "resource temporarily unavailable") ||
		strings.Contains(errStr, "input/output error")
}

func (d *Device) handleToRadio(data []byte) {
	// Parse ToRadio message (simplified)
	// Field 3 (want_config_id) is the config request
	pos := 0
	for pos < len(data) {
		if pos >= len(data) {
			break
		}
		tag := data[pos]
		fieldNum := tag >> 3
		wireType := tag & 0x07
		pos++

		switch wireType {
		case 0: // Varint
			val, n := decodeVarint(data[pos:])
			pos += n

			if fieldNum == 3 && val > 0 {
				// Config request
				d.logger("Received config request (id=%d)", val)
				d.sendConfig(uint32(val))
			}
		case 2: // Length-delimited
			length, n := decodeVarint(data[pos:])
			pos += n
			pos += int(length)
		}
	}
}

func (d *Device) sendConfig(configID uint32) {
	d.mu.Lock()
	if d.configSent {
		d.mu.Unlock()
		return
	}
	d.configSent = true
	d.mu.Unlock()

	d.logger("Sending configuration")

	// Send MyInfo
	myInfo := EncodeMyNodeInfo(d.config.NodeNum, 1)
	_ = d.sendFromRadio(nil, myInfo, nil, 0)

	// Send our own NodeInfo
	user := EncodeUser(
		fmt.Sprintf("!%08x", d.config.NodeNum),
		d.config.LongName,
		d.config.ShortName,
		d.config.HWModel,
	)
	position := EncodePosition(
		int32(d.config.Latitude*1e7),
		int32(d.config.Longitude*1e7),
		d.config.Altitude,
		uint32(time.Now().Unix()),
	)
	nodeInfo := EncodeNodeInfo(d.config.NodeNum, user, position, 0, uint32(time.Now().Unix()))
	_ = d.sendFromRadio(nil, nil, nodeInfo, 0)

	// Send other nodes
	for _, node := range d.config.SimulatedNodes {
		user := EncodeUser(
			fmt.Sprintf("!%08x", node.NodeNum),
			node.LongName,
			node.ShortName,
			node.HWModel,
		)
		position := EncodePosition(
			int32(node.Latitude*1e7),
			int32(node.Longitude*1e7),
			node.Altitude,
			uint32(time.Now().Unix()-int64(rand.Intn(3600))),
		)
		nodeInfo := EncodeNodeInfo(
			node.NodeNum,
			user,
			position,
			float32(rand.Intn(20)-10),
			uint32(time.Now().Unix()-int64(rand.Intn(600))),
		)
		_ = d.sendFromRadio(nil, nil, nodeInfo, 0)
	}

	// Send config complete
	_ = d.sendFromRadio(nil, nil, nil, configID)

	d.logger("Configuration sent")
}

func (d *Device) sendFromRadio(packet, myInfo, nodeInfo []byte, configCompleteID uint32) error {
	d.packetID++
	fromRadio := EncodeFromRadio(d.packetID, packet, myInfo, nodeInfo, configCompleteID)

	d.logger("Sending FromRadio: %d bytes", len(fromRadio))
	return d.framer.WritePacket(fromRadio)
}

func (d *Device) createTextMessagePacket(fromNode uint32, text string) []byte {
	// Create Data message with text
	data := EncodeData(1, []byte(text)) // PortNum 1 = TEXT_MESSAGE_APP

	// Create MeshPacket
	packet := EncodeMeshPacket(
		fromNode,
		0xFFFFFFFF, // Broadcast
		0,          // Channel
		d.packetID,
		data,
		uint32(time.Now().Unix()),
		float32(rand.Intn(20)-5), // SNR
		int32(-60-rand.Intn(40)), // RSSI
		3,                        // HopLimit
	)

	return packet
}

func (d *Device) createPositionPacket(fromNode uint32, lat, lon float64, alt int32) []byte {
	// Create Position message
	position := EncodePosition(
		int32(lat*1e7),
		int32(lon*1e7),
		alt,
		uint32(time.Now().Unix()),
	)

	// Create Data message with position
	data := EncodeData(3, position) // PortNum 3 = POSITION_APP

	// Create MeshPacket
	packet := EncodeMeshPacket(
		fromNode,
		0xFFFFFFFF, // Broadcast
		0,          // Channel
		d.packetID,
		data,
		uint32(time.Now().Unix()),
		float32(rand.Intn(20)-5),
		int32(-60-rand.Intn(40)),
		3,
	)

	return packet
}

func (d *Device) messageLoop(ctx context.Context) {
	d.logger("Starting message loop (interval=%v)", d.config.MessageInterval)

	ticker := time.NewTicker(d.config.MessageInterval)
	defer ticker.Stop()

	messages := []string{
		"Hello from the mesh!",
		"Testing 1 2 3",
		"Meshtastic is awesome!",
		"Anyone copy?",
		"Good morning mesh!",
		"Signal check",
		"Weather is nice today",
		"73s de simulated node",
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stopCh:
			return
		case <-ticker.C:
			// Pick a random node and message
			if len(d.config.SimulatedNodes) > 0 {
				node := d.config.SimulatedNodes[rand.Intn(len(d.config.SimulatedNodes))]
				msg := messages[rand.Intn(len(messages))]

				d.logger("Sending message from %s: %s", node.ShortName, msg)
				_ = d.SendTextMessage(node.NodeNum, msg)
			}
		}
	}
}

func decodeVarint(data []byte) (uint64, int) {
	var val uint64
	var shift uint
	for i, b := range data {
		val |= uint64(b&0x7F) << shift
		if b&0x80 == 0 {
			return val, i + 1
		}
		shift += 7
		if shift >= 64 {
			return 0, 0
		}
	}
	return 0, 0
}

// WriteRaw writes raw bytes to the master end (for testing)
func (d *Device) WriteRaw(data []byte) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.pty == nil {
		return fmt.Errorf("device not started")
	}

	_, err := d.pty.Master.Write(data)
	return err
}

// ReadRaw reads raw bytes from the master end (for testing)
func (d *Device) ReadRaw(buf []byte) (int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.pty == nil {
		return 0, fmt.Errorf("device not started")
	}

	_ = d.pty.Master.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	return d.pty.Master.Read(buf)
}

// WriteFramedPacket writes a framed packet to the master end
func (d *Device) WriteFramedPacket(data []byte) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.framer == nil {
		return fmt.Errorf("device not started")
	}

	return d.framer.WritePacket(data)
}

// MasterReader returns an io.Reader for the master end
func (d *Device) MasterReader() io.Reader {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.pty != nil {
		return d.pty.Master
	}
	return nil
}

// MasterWriter returns an io.Writer for the master end
func (d *Device) MasterWriter() io.Writer {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.pty != nil {
		return d.pty.Master
	}
	return nil
}

// SetReadDeadline sets the read deadline on the master
func (d *Device) SetReadDeadline(t time.Time) error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.pty != nil {
		return d.pty.Master.SetReadDeadline(t)
	}
	return fmt.Errorf("device not started")
}

// Frame creates a framed packet from raw data
func Frame(data []byte) []byte {
	result := make([]byte, 4+len(data))
	result[0] = meshtastic.Magic1
	result[1] = meshtastic.Magic2
	binary.BigEndian.PutUint16(result[2:4], uint16(len(data)))
	copy(result[4:], data)
	return result
}

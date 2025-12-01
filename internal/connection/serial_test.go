//go:build unix

package connection

import (
	"context"
	"testing"
	"time"

	"github.com/iamruinous/meshtastic-message-relay/internal/config"
	"github.com/iamruinous/meshtastic-message-relay/internal/logging"
	"github.com/iamruinous/meshtastic-message-relay/pkg/meshtastic/simulator"
)

func init() {
	// Initialize logging for tests
	_ = logging.Initialize(logging.Config{Level: "error", Format: "text"})
}

func TestSerialConnection(t *testing.T) {
	// Create a simulated device
	simConfig := simulator.DefaultConfig()
	simConfig.Verbose = testing.Verbose()
	simConfig.MessageInterval = 0 // Disable auto messages

	device := simulator.NewTestDevice(t)
	defer device.Stop()

	path := device.Start()
	t.Logf("Simulator started on %s", path)

	// Create serial connection config
	cfg := config.SerialConfig{
		Port: path,
		Baud: 115200,
	}

	// Create and connect
	conn, err := NewSerial(cfg)
	if err != nil {
		t.Fatalf("Failed to create serial connection: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = conn.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Verify connection
	if !conn.IsConnected() {
		t.Error("Connection should be connected")
	}

	if conn.Name() != "serial:"+path {
		t.Errorf("Expected name 'serial:%s', got '%s'", path, conn.Name())
	}

	// Wait for config to be exchanged
	if !device.WaitForConfig(5 * time.Second) {
		t.Error("Config was not requested/sent within timeout")
	}

	t.Log("Config exchange completed")
}

func TestSerialReceiveMessage(t *testing.T) {
	// Create a simulated device
	device := simulator.NewTestDevice(t)
	defer device.Stop()

	path := device.Start()

	// Create and connect
	cfg := config.SerialConfig{
		Port: path,
		Baud: 115200,
	}

	conn, err := NewSerial(cfg)
	if err != nil {
		t.Fatalf("Failed to create serial connection: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = conn.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Wait for config
	if !device.WaitForConfig(5 * time.Second) {
		t.Fatal("Config was not sent")
	}

	// Give some time for config messages to be processed
	time.Sleep(200 * time.Millisecond)

	// Send a text message from the simulator
	testMessage := "Hello from test!"
	fromNode := device.Device.Config().SimulatedNodes[0].NodeNum

	device.MustSendTextMessage(fromNode, testMessage)

	// Wait for the message
	select {
	case msg := <-conn.Messages():
		if msg == nil {
			t.Fatal("Received nil message")
		}
		t.Logf("Received message from !%08x: %+v", msg.From, msg.Payload)

		if msg.From != fromNode {
			t.Errorf("Expected from !%08x, got !%08x", fromNode, msg.From)
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

func TestSerialMultipleMessages(t *testing.T) {
	device := simulator.NewTestDevice(t)
	defer device.Stop()

	path := device.Start()

	cfg := config.SerialConfig{
		Port: path,
		Baud: 115200,
	}

	conn, err := NewSerial(cfg)
	if err != nil {
		t.Fatalf("Failed to create serial connection: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err = conn.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Wait for config
	device.WaitForConfig(5 * time.Second)
	time.Sleep(200 * time.Millisecond)

	// Send multiple messages
	messages := []string{"Message 1", "Message 2", "Message 3"}
	nodes := device.Device.Config().SimulatedNodes

	for i, msg := range messages {
		node := nodes[i%len(nodes)]
		device.MustSendTextMessage(node.NodeNum, msg)
		time.Sleep(50 * time.Millisecond)
	}

	// Receive messages
	received := 0
	timeout := time.After(5 * time.Second)

	for received < len(messages) {
		select {
		case msg := <-conn.Messages():
			if msg != nil {
				t.Logf("Received message %d: %+v", received+1, msg.Payload)
				received++
			}
		case <-timeout:
			t.Fatalf("Only received %d/%d messages", received, len(messages))
		}
	}

	t.Logf("Successfully received all %d messages", received)
}

func TestSerialClose(t *testing.T) {
	device := simulator.NewTestDevice(t)
	defer device.Stop()

	path := device.Start()

	cfg := config.SerialConfig{
		Port: path,
		Baud: 115200,
	}

	conn, err := NewSerial(cfg)
	if err != nil {
		t.Fatalf("Failed to create serial connection: %v", err)
	}

	ctx := context.Background()
	err = conn.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Close the connection
	err = conn.Close()
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}

	// Verify it's disconnected
	if conn.IsConnected() {
		t.Error("Connection should be disconnected after Close")
	}

	// Messages channel should be closed
	select {
	case _, ok := <-conn.Messages():
		if ok {
			t.Error("Messages channel should be closed")
		}
	default:
		// Channel might be empty but not closed yet, that's ok
	}
}

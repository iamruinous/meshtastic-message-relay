package simulator

import (
	"context"
	"testing"
	"time"
)

// TestDevice is a helper for testing with a simulated device
type TestDevice struct {
	*Device
	t      *testing.T
	ctx    context.Context
	cancel context.CancelFunc
}

// NewTestDevice creates a new test device with default configuration
func NewTestDevice(t *testing.T) *TestDevice {
	config := DefaultConfig()
	config.Verbose = testing.Verbose()
	config.MessageInterval = 0 // Disable auto messages for tests

	ctx, cancel := context.WithCancel(context.Background())

	return &TestDevice{
		Device: New(config),
		t:      t,
		ctx:    ctx,
		cancel: cancel,
	}
}

// NewTestDeviceWithConfig creates a test device with custom configuration
func NewTestDeviceWithConfig(t *testing.T, config DeviceConfig) *TestDevice {
	ctx, cancel := context.WithCancel(context.Background())

	return &TestDevice{
		Device: New(config),
		t:      t,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start starts the test device and returns the path
func (td *TestDevice) Start() string {
	path, err := td.Device.Start(td.ctx)
	if err != nil {
		td.t.Fatalf("Failed to start test device: %v", err)
	}
	// Give it a moment to initialize
	time.Sleep(50 * time.Millisecond)
	return path
}

// Stop stops the test device
func (td *TestDevice) Stop() {
	td.cancel()
	_ = td.Device.Stop()
}

// MustSendTextMessage sends a text message or fails the test
func (td *TestDevice) MustSendTextMessage(fromNode uint32, text string) {
	if err := td.SendTextMessage(fromNode, text); err != nil {
		td.t.Fatalf("Failed to send text message: %v", err)
	}
}

// MustSendPosition sends a position update or fails the test
func (td *TestDevice) MustSendPosition(fromNode uint32, lat, lon float64, alt int32) {
	if err := td.SendPosition(fromNode, lat, lon, alt); err != nil {
		td.t.Fatalf("Failed to send position: %v", err)
	}
}

// WaitForConfig waits for configuration to be requested and sent
func (td *TestDevice) WaitForConfig(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		td.mu.RLock()
		sent := td.configSent
		td.mu.RUnlock()
		if sent {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// Context returns the test context
func (td *TestDevice) Context() context.Context {
	return td.ctx
}

package meshtastic

import (
	"bytes"
	"testing"
)

func TestStreamFramerWriteRead(t *testing.T) {
	// Create a buffer to act as the stream
	buf := &bytes.Buffer{}

	framer := NewStreamFramer(buf, buf)

	// Test data
	testData := []byte("Hello, Meshtastic!")

	// Write a packet
	err := framer.WritePacket(testData)
	if err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	// Read the packet back
	readData, err := framer.ReadPacket()
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}

	// Verify
	if !bytes.Equal(testData, readData) {
		t.Errorf("Data mismatch: expected %v, got %v", testData, readData)
	}
}

func TestStreamFramerMultiplePackets(t *testing.T) {
	buf := &bytes.Buffer{}
	framer := NewStreamFramer(buf, buf)

	packets := [][]byte{
		[]byte("Packet 1"),
		[]byte("Packet 2 with more data"),
		[]byte("P3"),
		make([]byte, 100), // Empty packet
	}

	// Write all packets
	for i, data := range packets {
		if err := framer.WritePacket(data); err != nil {
			t.Fatalf("WritePacket %d failed: %v", i, err)
		}
	}

	// Read all packets back
	for i, expected := range packets {
		data, err := framer.ReadPacket()
		if err != nil {
			t.Fatalf("ReadPacket %d failed: %v", i, err)
		}
		if !bytes.Equal(expected, data) {
			t.Errorf("Packet %d mismatch: expected %v, got %v", i, expected, data)
		}
	}
}

func TestStreamFramerInvalidMagic(t *testing.T) {
	// Create buffer with invalid magic bytes
	buf := bytes.NewBuffer([]byte{0x00, 0x00, 0x00, 0x04, 't', 'e', 's', 't'})

	framer := NewStreamFramer(buf, nil)

	_, err := framer.ReadPacket()
	if err != ErrInvalidMagic {
		t.Errorf("Expected ErrInvalidMagic, got %v", err)
	}
}

func TestStreamFramerPacketTooLarge(t *testing.T) {
	// Create a framer
	buf := &bytes.Buffer{}
	framer := NewStreamFramer(buf, buf)

	// Try to write a packet that's too large
	largeData := make([]byte, MaxPacketSize+1)
	err := framer.WritePacket(largeData)
	if err != ErrPacketTooLarge {
		t.Errorf("Expected ErrPacketTooLarge, got %v", err)
	}
}

func TestFrameFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	framer := NewStreamFramer(buf, buf)

	data := []byte("test")
	_ = framer.WritePacket(data)

	// Check the raw bytes
	raw := buf.Bytes()

	// Should be: Magic1, Magic2, Length(2 bytes BE), Data
	if raw[0] != Magic1 {
		t.Errorf("Expected Magic1 (0x%02x), got 0x%02x", Magic1, raw[0])
	}
	if raw[1] != Magic2 {
		t.Errorf("Expected Magic2 (0x%02x), got 0x%02x", Magic2, raw[1])
	}

	// Length should be 4 (big endian)
	if raw[2] != 0x00 || raw[3] != 0x04 {
		t.Errorf("Expected length 0x0004, got 0x%02x%02x", raw[2], raw[3])
	}

	// Data should follow
	if !bytes.Equal(raw[4:], data) {
		t.Errorf("Data mismatch: expected %v, got %v", data, raw[4:])
	}
}

func TestSyncToMagic(t *testing.T) {
	// Create buffer with garbage followed by a valid packet
	garbage := []byte{0x00, 0x01, 0x02, 0x03, 0xFF}
	validPacket := []byte{Magic1, Magic2, 0x00, 0x04, 't', 'e', 's', 't'}

	buf := bytes.NewBuffer(append(garbage, validPacket...))
	framer := NewStreamFramer(buf, nil)

	// Sync should find the magic bytes
	err := framer.SyncToMagic()
	if err != nil {
		t.Fatalf("SyncToMagic failed: %v", err)
	}

	// Now we should be able to read the length and data
	// (the magic bytes were consumed by SyncToMagic)
	remaining := buf.Bytes()
	if len(remaining) != 6 { // length (2) + data (4)
		t.Errorf("Expected 6 bytes remaining, got %d", len(remaining))
	}
}

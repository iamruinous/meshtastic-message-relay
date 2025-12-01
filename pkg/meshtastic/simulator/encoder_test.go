package simulator

import (
	"testing"

	"github.com/iamruinous/meshtastic-message-relay/pkg/meshtastic"
)

func TestEncodeVarint(t *testing.T) {
	tests := []struct {
		value    uint64
		expected []byte
	}{
		{0, []byte{0x00}},
		{1, []byte{0x01}},
		{127, []byte{0x7F}},
		{128, []byte{0x80, 0x01}},
		{300, []byte{0xAC, 0x02}},
		{16384, []byte{0x80, 0x80, 0x01}},
	}

	for _, tt := range tests {
		result := encodeVarint(tt.value)
		if len(result) != len(tt.expected) {
			t.Errorf("encodeVarint(%d): expected length %d, got %d", tt.value, len(tt.expected), len(result))
			continue
		}
		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("encodeVarint(%d): byte %d expected 0x%02X, got 0x%02X", tt.value, i, tt.expected[i], result[i])
			}
		}
	}
}

func TestEncodeMyNodeInfo(t *testing.T) {
	nodeNum := uint32(0x12345678)
	rebootCount := uint32(5)

	data := EncodeMyNodeInfo(nodeNum, rebootCount)

	// Parse it back using the meshtastic package
	result, err := meshtastic.ParseFromRadio(EncodeFromRadio(1, nil, data, nil, 0))
	if err != nil {
		t.Fatalf("Failed to parse encoded MyNodeInfo: %v", err)
	}

	if result.MyInfo == nil {
		t.Fatal("MyInfo is nil")
	}

	if result.MyInfo.MyNodeNum != nodeNum {
		t.Errorf("Expected node num %d, got %d", nodeNum, result.MyInfo.MyNodeNum)
	}

	if result.MyInfo.RebootCount != rebootCount {
		t.Errorf("Expected reboot count %d, got %d", rebootCount, result.MyInfo.RebootCount)
	}
}

func TestEncodeUser(t *testing.T) {
	id := "!12345678"
	longName := "Test Node"
	shortName := "TST1"
	hwModel := uint32(9) // TBEAM

	data := EncodeUser(id, longName, shortName, hwModel)

	// This should be valid protobuf that can be nested in NodeInfo
	if len(data) == 0 {
		t.Error("Encoded user is empty")
	}

	// Verify it contains our data by checking for substrings
	// (rough check since we're encoding as length-delimited strings)
	found := false
	for i := 0; i < len(data)-len(longName)+1; i++ {
		if string(data[i:i+len(longName)]) == longName {
			found = true
			break
		}
	}
	if !found {
		t.Error("Long name not found in encoded data")
	}
}

func TestEncodeNodeInfo(t *testing.T) {
	nodeNum := uint32(0xAABBCCDD)
	user := EncodeUser("!aabbccdd", "Remote Node", "REM1", 9)
	position := EncodePosition(377749000, -1224194000, 100, 1700000000)
	snr := float32(5.5)
	lastHeard := uint32(1700000000)

	data := EncodeNodeInfo(nodeNum, user, position, snr, lastHeard)

	// Parse it back
	fromRadio := EncodeFromRadio(1, nil, nil, data, 0)
	result, err := meshtastic.ParseFromRadio(fromRadio)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if result.NodeInfo == nil {
		t.Fatal("NodeInfo is nil")
	}

	if result.NodeInfo.Num != nodeNum {
		t.Errorf("Expected node num 0x%08X, got 0x%08X", nodeNum, result.NodeInfo.Num)
	}

	if result.NodeInfo.User == nil {
		t.Error("User is nil")
	} else {
		if result.NodeInfo.User.LongName != "Remote Node" {
			t.Errorf("Expected long name 'Remote Node', got '%s'", result.NodeInfo.User.LongName)
		}
	}
}

func TestEncodeMeshPacket(t *testing.T) {
	from := uint32(0x11111111)
	to := uint32(0xFFFFFFFF)
	channel := uint32(0)
	id := uint32(12345)
	rxTime := uint32(1700000000)
	snr := float32(-2.5)
	rssi := int32(-80)
	hopLimit := uint32(3)

	// Create a text message
	textData := EncodeData(1, []byte("Hello World"))
	packet := EncodeMeshPacket(from, to, channel, id, textData, rxTime, snr, rssi, hopLimit)

	// Parse it back
	fromRadio := EncodeFromRadio(1, packet, nil, nil, 0)
	result, err := meshtastic.ParseFromRadio(fromRadio)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if result.Packet == nil {
		t.Fatal("Packet is nil")
	}

	if result.Packet.From != from {
		t.Errorf("Expected from 0x%08X, got 0x%08X", from, result.Packet.From)
	}

	if result.Packet.To != to {
		t.Errorf("Expected to 0x%08X, got 0x%08X", to, result.Packet.To)
	}

	if result.Packet.ID != id {
		t.Errorf("Expected id %d, got %d", id, result.Packet.ID)
	}

	if result.Packet.Decoded == nil {
		t.Fatal("Decoded is nil")
	}

	if result.Packet.Decoded.PortNum != meshtastic.PortNumTextMessageApp {
		t.Errorf("Expected port num %d, got %d", meshtastic.PortNumTextMessageApp, result.Packet.Decoded.PortNum)
	}

	if string(result.Packet.Decoded.Payload) != "Hello World" {
		t.Errorf("Expected payload 'Hello World', got '%s'", string(result.Packet.Decoded.Payload))
	}
}

func TestEncodeFromRadio(t *testing.T) {
	id := uint32(100)
	configCompleteID := uint32(1)

	data := EncodeFromRadio(id, nil, nil, nil, configCompleteID)

	result, err := meshtastic.ParseFromRadio(data)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if result.ID != id {
		t.Errorf("Expected ID %d, got %d", id, result.ID)
	}

	if result.ConfigCompleteID != configCompleteID {
		t.Errorf("Expected ConfigCompleteID %d, got %d", configCompleteID, result.ConfigCompleteID)
	}
}

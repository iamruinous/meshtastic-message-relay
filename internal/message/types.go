package message

import "time"

// PortNum represents the Meshtastic application port number.
type PortNum int32

// Common Meshtastic port numbers.
const (
	PortNumUnknown       PortNum = 0
	PortNumTextMessage   PortNum = 1
	PortNumRemoteHW      PortNum = 2
	PortNumPosition      PortNum = 3
	PortNumNodeInfo      PortNum = 4
	PortNumRouting       PortNum = 5
	PortNumAdmin         PortNum = 6
	PortNumTextMessageCompressed PortNum = 7
	PortNumWaypoint      PortNum = 8
	PortNumAudio         PortNum = 9
	PortNumDetectionSensor PortNum = 10
	PortNumReply         PortNum = 32
	PortNumIPTunnel      PortNum = 33
	PortNumPaxCounter    PortNum = 34
	PortNumSerial        PortNum = 64
	PortNumStoreForward  PortNum = 65
	PortNumRangeTest     PortNum = 66
	PortNumTelemetry     PortNum = 67
	PortNumZPS           PortNum = 68
	PortNumSimulator     PortNum = 69
	PortNumTraceroute    PortNum = 70
	PortNumNeighborInfo  PortNum = 71
	PortNumAAtak         PortNum = 72
	PortNumMapReport     PortNum = 73
	PortNumPrivate       PortNum = 256
	PortNumAtakForwarder PortNum = 257
	PortNumMax           PortNum = 511
)

// String returns the string representation of the port number.
func (p PortNum) String() string {
	switch p {
	case PortNumTextMessage:
		return "TEXT_MESSAGE_APP"
	case PortNumPosition:
		return "POSITION_APP"
	case PortNumNodeInfo:
		return "NODEINFO_APP"
	case PortNumRouting:
		return "ROUTING_APP"
	case PortNumWaypoint:
		return "WAYPOINT_APP"
	case PortNumTelemetry:
		return "TELEMETRY_APP"
	case PortNumTraceroute:
		return "TRACEROUTE_APP"
	case PortNumNeighborInfo:
		return "NEIGHBORINFO_APP"
	default:
		return "UNKNOWN_APP"
	}
}

// Packet represents a decoded Meshtastic packet.
type Packet struct {
	// ID is the unique packet identifier.
	ID uint32 `json:"id"`

	// From is the sender's node number.
	From uint32 `json:"from"`

	// To is the recipient's node number (0xFFFFFFFF for broadcast).
	To uint32 `json:"to"`

	// Channel is the channel index.
	Channel uint32 `json:"channel"`

	// PortNum indicates the application type.
	PortNum PortNum `json:"port_num"`

	// Payload is the decoded message payload.
	Payload interface{} `json:"payload"`

	// RawPayload is the raw bytes of the payload.
	RawPayload []byte `json:"raw_payload,omitempty"`

	// SNR is the signal-to-noise ratio.
	SNR float32 `json:"snr,omitempty"`

	// RSSI is the received signal strength indicator.
	RSSI int32 `json:"rssi,omitempty"`

	// HopLimit is the remaining hop count.
	HopLimit uint32 `json:"hop_limit,omitempty"`

	// WantAck indicates if an acknowledgment is requested.
	WantAck bool `json:"want_ack,omitempty"`

	// ReceivedAt is when the packet was received.
	ReceivedAt time.Time `json:"received_at"`

	// FromNode contains information about the sender (if known).
	FromNode *NodeInfo `json:"from_node,omitempty"`
}

// NodeInfo contains information about a mesh node.
type NodeInfo struct {
	// Num is the node number.
	Num uint32 `json:"num"`

	// User contains the user information.
	User *User `json:"user,omitempty"`

	// Position contains the last known position.
	Position *Position `json:"position,omitempty"`

	// LastHeard is when the node was last heard.
	LastHeard time.Time `json:"last_heard,omitempty"`

	// SNR is the signal-to-noise ratio when last heard.
	SNR float32 `json:"snr,omitempty"`
}

// User contains user information for a node.
type User struct {
	// ID is the user ID (usually based on MAC address).
	ID string `json:"id"`

	// LongName is the long display name.
	LongName string `json:"long_name"`

	// ShortName is the short display name (4 chars max).
	ShortName string `json:"short_name"`

	// HWModel is the hardware model identifier.
	HWModel string `json:"hw_model,omitempty"`
}

// Position contains GPS position information.
type Position struct {
	// Latitude in degrees.
	Latitude float64 `json:"latitude"`

	// Longitude in degrees.
	Longitude float64 `json:"longitude"`

	// Altitude in meters above sea level.
	Altitude int32 `json:"altitude,omitempty"`

	// Time is when the position was recorded.
	Time time.Time `json:"time,omitempty"`
}

// TextMessage represents a decoded text message.
type TextMessage struct {
	// Text is the message content.
	Text string `json:"text"`
}

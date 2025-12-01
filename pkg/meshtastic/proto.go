package meshtastic

import (
	"encoding/binary"
	"errors"
	"math"
	"time"
)

// PortNum represents Meshtastic application port numbers
type PortNum uint32

// Meshtastic port numbers
const (
	PortNumUnknownApp         PortNum = 0
	PortNumTextMessageApp     PortNum = 1
	PortNumRemoteHardwareApp  PortNum = 2
	PortNumPositionApp        PortNum = 3
	PortNumNodeInfoApp        PortNum = 4
	PortNumRoutingApp         PortNum = 5
	PortNumAdminApp           PortNum = 6
	PortNumTextMsgCompressApp PortNum = 7
	PortNumWaypointApp        PortNum = 8
	PortNumAudioApp           PortNum = 9
	PortNumDetectionSensorApp PortNum = 10
	PortNumReplyApp           PortNum = 32
	PortNumIPTunnelApp        PortNum = 33
	PortNumPaxcounterApp      PortNum = 34
	PortNumSerialApp          PortNum = 64
	PortNumStoreForwardApp    PortNum = 65
	PortNumRangeTestApp       PortNum = 66
	PortNumTelemetryApp       PortNum = 67
	PortNumZpsApp             PortNum = 68
	PortNumSimulatorApp       PortNum = 69
	PortNumTracerouteApp      PortNum = 70
	PortNumNeighborinfoApp    PortNum = 71
	PortNumAtakPlugin         PortNum = 72
	PortNumMapReportApp       PortNum = 73
	PortNumPrivateApp         PortNum = 256
	PortNumAtakForwarder      PortNum = 257
	PortNumMax                PortNum = 511
)

// String returns the string representation of the port number
func (p PortNum) String() string {
	names := map[PortNum]string{
		PortNumUnknownApp:         "UNKNOWN_APP",
		PortNumTextMessageApp:     "TEXT_MESSAGE_APP",
		PortNumRemoteHardwareApp:  "REMOTE_HARDWARE_APP",
		PortNumPositionApp:        "POSITION_APP",
		PortNumNodeInfoApp:        "NODEINFO_APP",
		PortNumRoutingApp:         "ROUTING_APP",
		PortNumAdminApp:           "ADMIN_APP",
		PortNumTextMsgCompressApp: "TEXT_MESSAGE_COMPRESSED_APP",
		PortNumWaypointApp:        "WAYPOINT_APP",
		PortNumAudioApp:           "AUDIO_APP",
		PortNumDetectionSensorApp: "DETECTION_SENSOR_APP",
		PortNumReplyApp:           "REPLY_APP",
		PortNumIPTunnelApp:        "IP_TUNNEL_APP",
		PortNumPaxcounterApp:      "PAXCOUNTER_APP",
		PortNumSerialApp:          "SERIAL_APP",
		PortNumStoreForwardApp:    "STORE_FORWARD_APP",
		PortNumRangeTestApp:       "RANGE_TEST_APP",
		PortNumTelemetryApp:       "TELEMETRY_APP",
		PortNumZpsApp:             "ZPS_APP",
		PortNumSimulatorApp:       "SIMULATOR_APP",
		PortNumTracerouteApp:      "TRACEROUTE_APP",
		PortNumNeighborinfoApp:    "NEIGHBORINFO_APP",
		PortNumAtakPlugin:         "ATAK_PLUGIN",
		PortNumMapReportApp:       "MAP_REPORT_APP",
		PortNumPrivateApp:         "PRIVATE_APP",
		PortNumAtakForwarder:      "ATAK_FORWARDER",
	}
	if name, ok := names[p]; ok {
		return name
	}
	return "UNKNOWN_APP"
}

// MeshPacket represents a decoded Meshtastic mesh packet
type MeshPacket struct {
	From         uint32
	To           uint32
	Channel      uint32
	ID           uint32
	RxTime       uint32
	RxSnr        float32
	RxRssi       int32
	HopLimit     uint32
	HopStart     uint32
	WantAck      bool
	Priority     uint32
	Decoded      *Data
	Encrypted    []byte
	PublicKey    []byte
	PkiEncrypted bool
}

// Data represents the decoded payload of a mesh packet
type Data struct {
	PortNum      PortNum
	Payload      []byte
	WantResponse bool
	Dest         uint32
	Source       uint32
	RequestID    uint32
	ReplyID      uint32
	Emoji        uint32
}

// FromRadio represents a message from the radio to the client
type FromRadio struct {
	ID                     uint32
	Packet                 *MeshPacket
	MyInfo                 *MyNodeInfo
	NodeInfo               *NodeInfo
	ConfigCompleteID       uint32
	Rebooted               bool
	Channel                *ChannelSettings
	QueueStatus            *QueueStatus
	XmodemPacket           []byte
	Metadata               *DeviceMetadata
	MqttClientProxyMessage *MqttClientProxyMessage
}

// ToRadio represents a message from the client to the radio
type ToRadio struct {
	Packet                 *MeshPacket
	WantConfigID           uint32
	Disconnect             bool
	XmodemPacket           []byte
	MqttClientProxyMessage *MqttClientProxyMessage
}

// MyNodeInfo contains information about this node
type MyNodeInfo struct {
	MyNodeNum      uint32
	RebootCount    uint32
	MinAppVersion  uint32
	DeviceMetadata *DeviceMetadata
}

// NodeInfo contains information about a node in the mesh
type NodeInfo struct {
	Num           uint32
	User          *User
	Position      *Position
	Snr           float32
	LastHeard     uint32
	DeviceMetrics *DeviceMetrics
	Channel       uint32
	ViaMqtt       bool
	Hops          uint32
	IsFavorite    bool
}

// User contains user information for a node
type User struct {
	ID         string
	LongName   string
	ShortName  string
	MacAddr    []byte
	HwModel    uint32
	IsLicensed bool
	Role       uint32
	PublicKey  []byte
}

// Position contains GPS position information
type Position struct {
	LatitudeI       int32
	LongitudeI      int32
	Altitude        int32
	Time            uint32
	LocationSource  uint32
	AltitudeSource  uint32
	Timestamp       uint32
	TimestampMillis int32
	AltitudeHae     int32
	AltGeoSep       int32
	PDOP            uint32
	HDOP            uint32
	VDOP            uint32
	GpsAccuracy     uint32
	GroundSpeed     uint32
	GroundTrack     uint32
	FixQuality      uint32
	FixType         uint32
	SatsInView      uint32
	SensorID        uint32
	NextUpdate      uint32
	SeqNumber       uint32
	PrecisionBits   uint32
}

// Latitude returns the latitude in degrees
func (p *Position) Latitude() float64 {
	return float64(p.LatitudeI) * 1e-7
}

// Longitude returns the longitude in degrees
func (p *Position) Longitude() float64 {
	return float64(p.LongitudeI) * 1e-7
}

// DeviceMetrics contains device telemetry
type DeviceMetrics struct {
	BatteryLevel       uint32
	Voltage            float32
	ChannelUtilization float32
	AirUtilTx          float32
	UptimeSeconds      uint32
}

// ChannelSettings contains channel configuration
type ChannelSettings struct {
	Index    uint32
	Settings *ChannelConfig
	Role     uint32
}

// ChannelConfig contains channel parameters
type ChannelConfig struct {
	ChannelNum      uint32
	Psk             []byte
	Name            string
	ID              uint32
	UplinkEnabled   bool
	DownlinkEnabled bool
	ModuleSettings  []byte
}

// QueueStatus contains queue status information
type QueueStatus struct {
	Res          int32
	Free         uint32
	MaxLen       uint32
	MeshPacketID uint32
}

// DeviceMetadata contains device information
type DeviceMetadata struct {
	FirmwareVersion    string
	DeviceStateVersion uint32
	CanShutdown        bool
	HasWifi            bool
	HasBluetooth       bool
	HasEthernet        bool
	Role               uint32
	PositionFlags      uint32
	HwModel            uint32
	HasRemoteHardware  bool
}

// MqttClientProxyMessage for MQTT proxy communication
type MqttClientProxyMessage struct {
	Topic    string
	Data     []byte
	Retained bool
}

// Errors for parsing
var (
	ErrInvalidProtobuf = errors.New("invalid protobuf data")
	ErrUnsupportedType = errors.New("unsupported message type")
)

// ParseFromRadio parses a FromRadio message from protobuf bytes
// This is a simplified parser - for production use, generate from .proto files
func ParseFromRadio(data []byte) (*FromRadio, error) {
	if len(data) < 2 {
		return nil, ErrInvalidProtobuf
	}

	fr := &FromRadio{}
	pos := 0

	for pos < len(data) {
		if pos >= len(data) {
			break
		}

		// Read field tag
		tag := data[pos]
		fieldNum := tag >> 3
		wireType := tag & 0x07
		pos++

		switch wireType {
		case 0: // Varint
			val, n := decodeVarint(data[pos:])
			pos += n
			switch fieldNum {
			case 1:
				fr.ID = uint32(val)
			case 8:
				fr.ConfigCompleteID = uint32(val)
			case 9:
				fr.Rebooted = val != 0
			}

		case 2: // Length-delimited
			length, n := decodeVarint(data[pos:])
			pos += n
			if pos+int(length) > len(data) {
				return nil, ErrInvalidProtobuf
			}
			fieldData := data[pos : pos+int(length)]
			pos += int(length)

			switch fieldNum {
			case 2: // packet
				packet, err := parseMeshPacket(fieldData)
				if err != nil {
					return nil, err
				}
				fr.Packet = packet
			case 3: // my_info
				myInfo, err := parseMyNodeInfo(fieldData)
				if err != nil {
					return nil, err
				}
				fr.MyInfo = myInfo
			case 4: // node_info
				nodeInfo, err := parseNodeInfo(fieldData)
				if err != nil {
					return nil, err
				}
				fr.NodeInfo = nodeInfo
			case 11: // xmodem_packet
				fr.XmodemPacket = fieldData
			}

		default:
			// Skip unknown wire types
			return nil, ErrUnsupportedType
		}
	}

	return fr, nil
}

func parseMeshPacket(data []byte) (*MeshPacket, error) {
	mp := &MeshPacket{}
	pos := 0

	for pos < len(data) {
		tag := data[pos]
		fieldNum := tag >> 3
		wireType := tag & 0x07
		pos++

		switch wireType {
		case 0: // Varint
			val, n := decodeVarint(data[pos:])
			pos += n
			switch fieldNum {
			case 1:
				mp.From = uint32(val)
			case 2:
				mp.To = uint32(val)
			case 3:
				mp.Channel = uint32(val)
			case 6:
				mp.ID = uint32(val)
			case 7:
				mp.RxTime = uint32(val)
			case 10:
				mp.HopLimit = uint32(val)
			case 11:
				mp.WantAck = val != 0
			case 12:
				mp.Priority = uint32(val)
			case 13:
				mp.RxSnr = float32(int32(val)) / 4.0
			case 15:
				mp.HopStart = uint32(val)
			case 17:
				mp.PkiEncrypted = val != 0
			}

		case 2: // Length-delimited
			length, n := decodeVarint(data[pos:])
			pos += n
			if pos+int(length) > len(data) {
				return nil, ErrInvalidProtobuf
			}
			fieldData := data[pos : pos+int(length)]
			pos += int(length)

			switch fieldNum {
			case 4: // decoded
				decoded, err := parseData(fieldData)
				if err != nil {
					return nil, err
				}
				mp.Decoded = decoded
			case 5: // encrypted
				mp.Encrypted = fieldData
			case 16: // public_key
				mp.PublicKey = fieldData
			}

		case 5: // 32-bit
			if pos+4 > len(data) {
				return nil, ErrInvalidProtobuf
			}
			val := binary.LittleEndian.Uint32(data[pos : pos+4])
			pos += 4
			if fieldNum == 14 {
				mp.RxRssi = int32(val)
			}
		}
	}

	return mp, nil
}

func parseData(data []byte) (*Data, error) {
	d := &Data{}
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
			switch fieldNum {
			case 1:
				d.PortNum = PortNum(val)
			case 3:
				d.WantResponse = val != 0
			case 4:
				d.Dest = uint32(val)
			case 5:
				d.Source = uint32(val)
			case 6:
				d.RequestID = uint32(val)
			case 7:
				d.ReplyID = uint32(val)
			case 8:
				d.Emoji = uint32(val)
			}

		case 2: // Length-delimited
			length, n := decodeVarint(data[pos:])
			pos += n
			if pos+int(length) > len(data) {
				return nil, ErrInvalidProtobuf
			}
			fieldData := data[pos : pos+int(length)]
			pos += int(length)

			if fieldNum == 2 {
				d.Payload = fieldData
			}
		}
	}

	return d, nil
}

//nolint:unparam // error return kept for API consistency with other parse functions
func parseMyNodeInfo(data []byte) (*MyNodeInfo, error) {
	info := &MyNodeInfo{}
	pos := 0

	for pos < len(data) {
		tag := data[pos]
		fieldNum := tag >> 3
		wireType := tag & 0x07
		pos++

		switch wireType {
		case 0: // Varint
			val, n := decodeVarint(data[pos:])
			pos += n
			switch fieldNum {
			case 1:
				info.MyNodeNum = uint32(val)
			case 8:
				info.RebootCount = uint32(val)
			case 11:
				info.MinAppVersion = uint32(val)
			}
		case 2: // Length-delimited
			length, n := decodeVarint(data[pos:])
			pos += n
			pos += int(length) // Skip for now
		}
	}

	return info, nil
}

func parseNodeInfo(data []byte) (*NodeInfo, error) {
	info := &NodeInfo{}
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
			switch fieldNum {
			case 1:
				info.Num = uint32(val)
			case 5:
				info.LastHeard = uint32(val)
			case 7:
				info.Channel = uint32(val)
			case 8:
				info.ViaMqtt = val != 0
			case 9:
				info.Hops = uint32(val)
			case 10:
				info.IsFavorite = val != 0
			}

		case 2: // Length-delimited
			length, n := decodeVarint(data[pos:])
			pos += n
			if pos+int(length) > len(data) {
				return nil, ErrInvalidProtobuf
			}
			fieldData := data[pos : pos+int(length)]
			pos += int(length)

			switch fieldNum {
			case 2:
				user, err := parseUser(fieldData)
				if err != nil {
					return nil, err
				}
				info.User = user
			case 3:
				position, err := parsePosition(fieldData)
				if err != nil {
					return nil, err
				}
				info.Position = position
			}

		case 5: // 32-bit
			if pos+4 > len(data) {
				return nil, ErrInvalidProtobuf
			}
			val := binary.LittleEndian.Uint32(data[pos : pos+4])
			pos += 4
			if fieldNum == 4 {
				info.Snr = float32FromBits(val)
			}
		}
	}

	return info, nil
}

func parseUser(data []byte) (*User, error) {
	user := &User{}
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
			switch fieldNum {
			case 5:
				user.HwModel = uint32(val)
			case 6:
				user.IsLicensed = val != 0
			case 7:
				user.Role = uint32(val)
			}

		case 2: // Length-delimited
			length, n := decodeVarint(data[pos:])
			pos += n
			if pos+int(length) > len(data) {
				return nil, ErrInvalidProtobuf
			}
			fieldData := data[pos : pos+int(length)]
			pos += int(length)

			switch fieldNum {
			case 1:
				user.ID = string(fieldData)
			case 2:
				user.LongName = string(fieldData)
			case 3:
				user.ShortName = string(fieldData)
			case 4:
				user.MacAddr = fieldData
			case 8:
				user.PublicKey = fieldData
			}
		}
	}

	return user, nil
}

func parsePosition(data []byte) (*Position, error) {
	pos := &Position{}
	offset := 0

	for offset < len(data) {
		if offset >= len(data) {
			break
		}
		tag := data[offset]
		fieldNum := tag >> 3
		wireType := tag & 0x07
		offset++

		switch wireType {
		case 0: // Varint
			val, n := decodeVarint(data[offset:])
			offset += n
			switch fieldNum {
			case 4:
				pos.Time = uint32(val)
			case 5:
				pos.LocationSource = uint32(val)
			case 6:
				pos.AltitudeSource = uint32(val)
			case 7:
				pos.Timestamp = uint32(val)
			case 14:
				pos.GroundSpeed = uint32(val)
			case 15:
				pos.GroundTrack = uint32(val)
			case 20:
				pos.SatsInView = uint32(val)
			}

		case 2: // Length-delimited
			length, n := decodeVarint(data[offset:])
			offset += n
			offset += int(length) // Skip

		case 5: // 32-bit (sfixed32, fixed32, float)
			if offset+4 > len(data) {
				return nil, ErrInvalidProtobuf
			}
			val := int32(binary.LittleEndian.Uint32(data[offset : offset+4]))
			offset += 4
			switch fieldNum {
			case 1:
				pos.LatitudeI = val
			case 2:
				pos.LongitudeI = val
			case 3:
				pos.Altitude = val
			}
		}
	}

	return pos, nil
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

func float32FromBits(bits uint32) float32 {
	return math.Float32frombits(bits)
}

// ToPacket converts a FromRadio message to our internal message format
func (fr *FromRadio) ToPacket() *Packet {
	if fr.Packet == nil {
		return nil
	}

	mp := fr.Packet
	p := &Packet{
		ID:         mp.ID,
		From:       mp.From,
		To:         mp.To,
		Channel:    mp.Channel,
		SNR:        mp.RxSnr,
		RSSI:       mp.RxRssi,
		HopLimit:   mp.HopLimit,
		WantAck:    mp.WantAck,
		ReceivedAt: time.Now(),
	}

	if mp.RxTime > 0 {
		p.ReceivedAt = time.Unix(int64(mp.RxTime), 0)
	}

	if mp.Decoded != nil {
		p.PortNum = mp.Decoded.PortNum
		p.RawPayload = mp.Decoded.Payload

		// Decode payload based on port number
		switch mp.Decoded.PortNum {
		case PortNumTextMessageApp:
			p.Payload = &TextMessage{Text: string(mp.Decoded.Payload)}
		case PortNumPositionApp:
			if pos, err := parsePosition(mp.Decoded.Payload); err == nil {
				p.Payload = pos
			}
		default:
			p.Payload = mp.Decoded.Payload
		}
	}

	return p
}

// Packet is our internal packet format
type Packet struct {
	ID         uint32
	From       uint32
	To         uint32
	Channel    uint32
	PortNum    PortNum
	Payload    interface{}
	RawPayload []byte
	SNR        float32
	RSSI       int32
	HopLimit   uint32
	WantAck    bool
	ReceivedAt time.Time
	FromNode   *NodeInfo
}

// TextMessage represents a text message payload
type TextMessage struct {
	Text string
}

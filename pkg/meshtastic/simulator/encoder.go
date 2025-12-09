package simulator

import (
	"encoding/binary"
	"math"
)

// Protobuf wire types
const (
	wireVarint = 0
	wire64bit  = 1
	wireBytes  = 2
	wire32bit  = 5
)

// encodeVarint encodes a uint64 as a protobuf varint
func encodeVarint(v uint64) []byte {
	var buf []byte
	for v >= 0x80 {
		buf = append(buf, byte(v)|0x80)
		v >>= 7
	}
	buf = append(buf, byte(v))
	return buf
}

// encodeTag encodes a protobuf field tag
func encodeTag(fieldNum, wireType int) []byte {
	return encodeVarint(uint64(fieldNum<<3 | wireType))
}

// encodeBytes encodes a length-delimited field
func encodeBytes(fieldNum int, data []byte) []byte {
	result := encodeTag(fieldNum, wireBytes)
	result = append(result, encodeVarint(uint64(len(data)))...)
	result = append(result, data...)
	return result
}

// encodeString encodes a string field
func encodeString(fieldNum int, s string) []byte {
	return encodeBytes(fieldNum, []byte(s))
}

// encodeUint32 encodes a uint32 varint field
func encodeUint32(fieldNum int, v uint32) []byte {
	result := encodeTag(fieldNum, wireVarint)
	result = append(result, encodeVarint(uint64(v))...)
	return result
}

// encodeInt32 encodes an int32 varint field (zigzag encoded for sint32)
func encodeInt32(fieldNum int, v int32) []byte {
	result := encodeTag(fieldNum, wireVarint)
	result = append(result, encodeVarint(uint64(v))...)
	return result
}

// encodeFixed32 encodes a fixed32 field
func encodeFixed32(fieldNum int, v uint32) []byte {
	result := encodeTag(fieldNum, wire32bit)
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, v)
	result = append(result, buf...)
	return result
}

// encodeSFixed32 encodes a sfixed32 field
func encodeSFixed32(fieldNum int, v int32) []byte {
	return encodeFixed32(fieldNum, uint32(v))
}

// encodeFloat32 encodes a float field
func encodeFloat32(fieldNum int, v float32) []byte {
	return encodeFixed32(fieldNum, math.Float32bits(v))
}

// EncodeMyNodeInfo encodes a MyNodeInfo message
func EncodeMyNodeInfo(nodeNum, rebootCount uint32) []byte {
	var msg []byte
	msg = append(msg, encodeUint32(1, nodeNum)...)     // my_node_num
	msg = append(msg, encodeUint32(8, rebootCount)...) // reboot_count
	msg = append(msg, encodeUint32(11, 30000)...)      // min_app_version
	return msg
}

// EncodeUser encodes a User message
func EncodeUser(id, longName, shortName string, hwModel uint32) []byte {
	var msg []byte
	msg = append(msg, encodeString(1, id)...)
	msg = append(msg, encodeString(2, longName)...)
	msg = append(msg, encodeString(3, shortName)...)
	msg = append(msg, encodeUint32(5, hwModel)...)
	return msg
}

// EncodePosition encodes a Position message
func EncodePosition(latitudeI, longitudeI, altitude int32, timestamp uint32) []byte {
	var msg []byte
	msg = append(msg, encodeSFixed32(1, latitudeI)...)
	msg = append(msg, encodeSFixed32(2, longitudeI)...)
	msg = append(msg, encodeSFixed32(3, altitude)...)
	if timestamp > 0 {
		msg = append(msg, encodeUint32(4, timestamp)...)
	}
	return msg
}

// EncodeNodeInfo encodes a NodeInfo message
func EncodeNodeInfo(num uint32, user, position []byte, snr float32, lastHeard uint32) []byte {
	var msg []byte
	msg = append(msg, encodeUint32(1, num)...)
	if len(user) > 0 {
		msg = append(msg, encodeBytes(2, user)...)
	}
	if len(position) > 0 {
		msg = append(msg, encodeBytes(3, position)...)
	}
	if snr != 0 {
		msg = append(msg, encodeFloat32(4, snr)...)
	}
	if lastHeard > 0 {
		msg = append(msg, encodeUint32(5, lastHeard)...)
	}
	return msg
}

// EncodeData encodes a Data message (decoded payload)
func EncodeData(portNum uint32, payload []byte) []byte {
	var msg []byte
	msg = append(msg, encodeUint32(1, portNum)...)
	msg = append(msg, encodeBytes(2, payload)...)
	return msg
}

// EncodeMeshPacket encodes a MeshPacket message
func EncodeMeshPacket(from, to, channel, id uint32, decoded []byte, rxTime uint32, snr float32, rssi int32, hopLimit uint32) []byte {
	var msg []byte
	msg = append(msg, encodeUint32(1, from)...)
	msg = append(msg, encodeUint32(2, to)...)
	msg = append(msg, encodeUint32(3, channel)...)
	if len(decoded) > 0 {
		msg = append(msg, encodeBytes(4, decoded)...)
	}
	msg = append(msg, encodeUint32(6, id)...)
	if rxTime > 0 {
		msg = append(msg, encodeUint32(7, rxTime)...)
	}
	msg = append(msg, encodeUint32(10, hopLimit)...)
	if snr != 0 {
		// SNR is stored as fixed point * 4
		snrVal := int32(snr * 4)
		msg = append(msg, encodeInt32(13, snrVal)...)
	}
	if rssi != 0 {
		msg = append(msg, encodeSFixed32(14, rssi)...)
	}
	return msg
}

// EncodeFromRadio encodes a FromRadio message
func EncodeFromRadio(id uint32, packet, myInfo, nodeInfo []byte, configCompleteID uint32) []byte {
	var msg []byte
	msg = append(msg, encodeUint32(1, id)...)
	if len(packet) > 0 {
		msg = append(msg, encodeBytes(2, packet)...)
	}
	if len(myInfo) > 0 {
		msg = append(msg, encodeBytes(3, myInfo)...)
	}
	if len(nodeInfo) > 0 {
		msg = append(msg, encodeBytes(4, nodeInfo)...)
	}
	if configCompleteID > 0 {
		msg = append(msg, encodeUint32(8, configCompleteID)...)
	}
	return msg
}

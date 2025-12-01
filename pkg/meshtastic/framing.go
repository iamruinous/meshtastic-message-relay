package meshtastic

import (
	"encoding/binary"
	"errors"
	"io"
)

// Protocol constants
const (
	// Magic bytes for serial/TCP protocol framing
	Magic1 byte = 0x94
	Magic2 byte = 0xc3

	// Maximum packet size
	MaxPacketSize = 512

	// Header size (2 magic bytes + 2 length bytes)
	HeaderSize = 4
)

var (
	// ErrInvalidMagic indicates invalid magic bytes in packet header
	ErrInvalidMagic = errors.New("invalid magic bytes")

	// ErrPacketTooLarge indicates packet exceeds maximum size
	ErrPacketTooLarge = errors.New("packet too large")

	// ErrIncompletePacket indicates incomplete packet data
	ErrIncompletePacket = errors.New("incomplete packet")
)

// StreamFramer handles framing/deframing of Meshtastic packets over streams
type StreamFramer struct {
	reader io.Reader
	writer io.Writer
	buffer []byte
}

// NewStreamFramer creates a new stream framer
func NewStreamFramer(r io.Reader, w io.Writer) *StreamFramer {
	return &StreamFramer{
		reader: r,
		writer: w,
		buffer: make([]byte, MaxPacketSize+HeaderSize),
	}
}

// ReadPacket reads a framed packet from the stream
func (f *StreamFramer) ReadPacket() ([]byte, error) {
	// Read header
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(f.reader, header); err != nil {
		return nil, err
	}

	// Validate magic bytes
	if header[0] != Magic1 || header[1] != Magic2 {
		return nil, ErrInvalidMagic
	}

	// Get length (big endian)
	length := binary.BigEndian.Uint16(header[2:4])

	if length > MaxPacketSize {
		return nil, ErrPacketTooLarge
	}

	// Read payload
	payload := make([]byte, length)
	if _, err := io.ReadFull(f.reader, payload); err != nil {
		return nil, err
	}

	return payload, nil
}

// WritePacket writes a framed packet to the stream
func (f *StreamFramer) WritePacket(data []byte) error {
	if len(data) > MaxPacketSize {
		return ErrPacketTooLarge
	}

	// Build header
	header := make([]byte, HeaderSize)
	header[0] = Magic1
	header[1] = Magic2
	binary.BigEndian.PutUint16(header[2:4], uint16(len(data)))

	// Write header
	if _, err := f.writer.Write(header); err != nil {
		return err
	}

	// Write payload
	if _, err := f.writer.Write(data); err != nil {
		return err
	}

	return nil
}

// SyncToMagic reads bytes until it finds the magic sequence
// Useful for recovering from stream corruption
func (f *StreamFramer) SyncToMagic() error {
	buf := make([]byte, 1)
	foundFirst := false

	for {
		if _, err := io.ReadFull(f.reader, buf); err != nil {
			return err
		}

		if foundFirst {
			if buf[0] == Magic2 {
				return nil
			}
			foundFirst = buf[0] == Magic1
		} else {
			foundFirst = buf[0] == Magic1
		}
	}
}

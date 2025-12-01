package meshtastic

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
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
	reader     io.Reader
	writer     io.Writer
	readBuffer []byte // Buffer for accumulating partial reads
	readPos    int    // Current position in read buffer
}

// NewStreamFramer creates a new stream framer
func NewStreamFramer(r io.Reader, w io.Writer) *StreamFramer {
	return &StreamFramer{
		reader:     r,
		writer:     w,
		readBuffer: make([]byte, MaxPacketSize+HeaderSize),
		readPos:    0,
	}
}

// ReadPacket reads a framed packet from the stream
// It handles timeouts gracefully by preserving partial reads across calls
func (f *StreamFramer) ReadPacket() ([]byte, error) {
	// Read until we have at least the header
	for f.readPos < HeaderSize {
		n, err := f.reader.Read(f.readBuffer[f.readPos:])
		if n > 0 {
			f.readPos += n
		}
		if err != nil {
			if isTemporaryError(err) && f.readPos > 0 {
				// We have partial data, keep trying
				continue
			}
			return nil, err
		}
	}

	// Validate magic bytes
	if f.readBuffer[0] != Magic1 || f.readBuffer[1] != Magic2 {
		// Invalid magic - discard first byte and try to resync
		copy(f.readBuffer, f.readBuffer[1:f.readPos])
		f.readPos--
		return nil, ErrInvalidMagic
	}

	// Get length (big endian)
	length := binary.BigEndian.Uint16(f.readBuffer[2:4])
	if length > MaxPacketSize {
		// Invalid length - reset buffer
		f.readPos = 0
		return nil, ErrPacketTooLarge
	}

	totalLen := HeaderSize + int(length)

	// Read until we have the full packet
	for f.readPos < totalLen {
		n, err := f.reader.Read(f.readBuffer[f.readPos:])
		if n > 0 {
			f.readPos += n
		}
		if err != nil {
			if isTemporaryError(err) && f.readPos < totalLen {
				// We have partial data, keep trying
				continue
			}
			return nil, err
		}
	}

	// Extract the payload
	payload := make([]byte, length)
	copy(payload, f.readBuffer[HeaderSize:totalLen])

	// Shift any remaining data to the beginning of the buffer
	remaining := f.readPos - totalLen
	if remaining > 0 {
		copy(f.readBuffer, f.readBuffer[totalLen:f.readPos])
	}
	f.readPos = remaining

	return payload, nil
}

// isTemporaryError checks if an error is temporary (timeout) and can be retried
func isTemporaryError(err error) bool {
	if err == nil {
		return false
	}

	// Check for net.Error timeout
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}

	// Check for os.ErrDeadlineExceeded
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}

	return false
}

// WritePacket writes a framed packet to the stream
func (f *StreamFramer) WritePacket(data []byte) error {
	if len(data) > MaxPacketSize {
		return ErrPacketTooLarge
	}

	// Build complete packet (header + payload) for atomic write
	packet := make([]byte, HeaderSize+len(data))
	packet[0] = Magic1
	packet[1] = Magic2
	binary.BigEndian.PutUint16(packet[2:4], uint16(len(data)))
	copy(packet[HeaderSize:], data)

	// Write entire packet in one call to ensure atomicity
	if _, err := f.writer.Write(packet); err != nil {
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

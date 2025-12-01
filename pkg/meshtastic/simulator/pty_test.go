//go:build unix

package simulator

import (
	"os"
	"testing"
	"time"

	"go.bug.st/serial"
)

func TestPTYBidirectional(t *testing.T) {
	// Create PTY
	pty, err := OpenPTY()
	if err != nil {
		t.Fatalf("Failed to create PTY: %v", err)
	}
	defer pty.Close()

	t.Logf("PTY created: master=%v, slave=%s", pty.Master.Fd(), pty.SlavePath)

	// Open slave side
	slave, err := os.OpenFile(pty.SlavePath, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("Failed to open slave: %v", err)
	}
	defer slave.Close()

	// Test master -> slave
	testData := []byte("Hello from master")
	n, err := pty.Master.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write to master: %v", err)
	}
	t.Logf("Wrote %d bytes to master", n)

	// Read from slave
	slave.SetReadDeadline(time.Now().Add(1 * time.Second))
	buf := make([]byte, 100)
	n, err = slave.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read from slave: %v", err)
	}
	t.Logf("Read %d bytes from slave: %s", n, string(buf[:n]))

	if string(buf[:n]) != string(testData) {
		t.Errorf("Master->Slave: expected %q, got %q", testData, buf[:n])
	}

	// Test slave -> master
	testData2 := []byte("Hello from slave")
	n, err = slave.Write(testData2)
	if err != nil {
		t.Fatalf("Failed to write to slave: %v", err)
	}
	t.Logf("Wrote %d bytes to slave", n)

	// Read from master
	pty.Master.SetReadDeadline(time.Now().Add(1 * time.Second))
	buf2 := make([]byte, 100)
	n, err = pty.Master.Read(buf2)
	if err != nil {
		t.Fatalf("Failed to read from master: %v", err)
	}
	t.Logf("Read %d bytes from master: %s", n, string(buf2[:n]))

	if string(buf2[:n]) != string(testData2) {
		t.Errorf("Slave->Master: expected %q, got %q", testData2, buf2[:n])
	}

	t.Log("Bidirectional PTY communication works!")
}

func TestPTYWithSerialLibrary(t *testing.T) {
	// Create PTY
	pty, err := OpenPTY()
	if err != nil {
		t.Fatalf("Failed to create PTY: %v", err)
	}
	defer pty.Close()

	t.Logf("PTY slave path: %s", pty.SlavePath)

	// Open slave using standard file (simulating what serial library does)
	slave, err := os.OpenFile(pty.SlavePath, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("Failed to open slave: %v", err)
	}
	defer slave.Close()

	// Set raw mode on slave too
	if err := setRawMode(int(slave.Fd())); err != nil {
		t.Logf("Warning: failed to set raw mode on slave: %v", err)
	}

	// Write framed packet from slave (simulating serial.requestConfig)
	framedPacket := []byte{0x94, 0xc3, 0x00, 0x02, 0x18, 0x01} // Magic + length(2) + WantConfig(2 bytes)
	n, err := slave.Write(framedPacket)
	if err != nil {
		t.Fatalf("Failed to write framed packet to slave: %v", err)
	}
	t.Logf("Wrote %d bytes to slave: %v", n, framedPacket)

	// Read from master
	pty.Master.SetReadDeadline(time.Now().Add(1 * time.Second))
	buf := make([]byte, 100)
	n, err = pty.Master.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read from master: %v", err)
	}
	t.Logf("Read %d bytes from master: %v", n, buf[:n])

	// Verify we got the framed packet
	if n != len(framedPacket) {
		t.Errorf("Expected %d bytes, got %d", len(framedPacket), n)
	}

	for i := 0; i < n && i < len(framedPacket); i++ {
		if buf[i] != framedPacket[i] {
			t.Errorf("Byte %d: expected 0x%02x, got 0x%02x", i, framedPacket[i], buf[i])
		}
	}

	t.Log("Framed packet communication works!")
}

func TestPTYWithGoSerial(t *testing.T) {
	// Create PTY
	pty, err := OpenPTY()
	if err != nil {
		t.Fatalf("Failed to create PTY: %v", err)
	}
	defer pty.Close()

	t.Logf("PTY slave path: %s", pty.SlavePath)

	// Open slave using go.bug.st/serial library
	mode := &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(pty.SlavePath, mode)
	if err != nil {
		t.Fatalf("Failed to open serial port: %v", err)
	}
	defer port.Close()

	// Set read timeout
	if err := port.SetReadTimeout(100 * time.Millisecond); err != nil {
		t.Logf("Warning: failed to set read timeout: %v", err)
	}

	// Write framed packet from serial port
	framedPacket := []byte{0x94, 0xc3, 0x00, 0x02, 0x18, 0x01}
	n, err := port.Write(framedPacket)
	if err != nil {
		t.Fatalf("Failed to write to serial port: %v", err)
	}
	t.Logf("Wrote %d bytes via serial library: %v", n, framedPacket)

	// Small delay for data to be transmitted
	time.Sleep(50 * time.Millisecond)

	// Read from master
	pty.Master.SetReadDeadline(time.Now().Add(1 * time.Second))
	buf := make([]byte, 100)
	n, err = pty.Master.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read from master: %v", err)
	}
	t.Logf("Read %d bytes from master: %v", n, buf[:n])

	// Verify
	if n != len(framedPacket) {
		t.Errorf("Expected %d bytes, got %d", len(framedPacket), n)
	}

	for i := 0; i < n && i < len(framedPacket); i++ {
		if buf[i] != framedPacket[i] {
			t.Errorf("Byte %d: expected 0x%02x, got 0x%02x", i, framedPacket[i], buf[i])
		}
	}

	t.Log("Serial library -> PTY master works!")
}

func TestPTYFullFlow(t *testing.T) {
	// Create PTY
	pty, err := OpenPTY()
	if err != nil {
		t.Fatalf("Failed to create PTY: %v", err)
	}
	defer pty.Close()

	t.Logf("PTY slave path: %s", pty.SlavePath)

	// Open slave using serial library
	mode := &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(pty.SlavePath, mode)
	if err != nil {
		t.Fatalf("Failed to open serial port: %v", err)
	}
	defer port.Close()

	if err := port.SetReadTimeout(100 * time.Millisecond); err != nil {
		t.Logf("Warning: failed to set read timeout: %v", err)
	}

	// Start a read loop on the serial port (simulating serial.readLoop)
	serialReceived := make(chan []byte, 10)
	go func() {
		buf := make([]byte, 100)
		for {
			n, err := port.Read(buf)
			if err != nil {
				continue
			}
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				serialReceived <- data
			}
		}
	}()

	// Start a read loop on the master PTY (simulating simulator.readLoop)
	masterReceived := make(chan []byte, 10)
	go func() {
		buf := make([]byte, 100)
		for {
			pty.Master.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, err := pty.Master.Read(buf)
			if err != nil {
				continue
			}
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				masterReceived <- data
			}
		}
	}()

	// Wait a bit for goroutines to start
	time.Sleep(50 * time.Millisecond)

	// Simulate requestConfig: write from serial port after delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		framedPacket := []byte{0x94, 0xc3, 0x00, 0x02, 0x18, 0x01}
		n, err := port.Write(framedPacket)
		if err != nil {
			t.Errorf("Failed to write to serial port: %v", err)
			return
		}
		t.Logf("Wrote %d bytes via serial port: %v", n, framedPacket)
	}()

	// Wait for master to receive the packet
	select {
	case data := <-masterReceived:
		t.Logf("Master received: %v", data)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for master to receive data")
	}

	// Simulate simulator response: write from master
	responsePacket := []byte{0x94, 0xc3, 0x00, 0x04, 0xDE, 0xAD, 0xBE, 0xEF}
	n, err := pty.Master.Write(responsePacket)
	if err != nil {
		t.Fatalf("Failed to write to master: %v", err)
	}
	t.Logf("Wrote %d bytes via master: %v", n, responsePacket)

	// Wait for serial port to receive the response
	select {
	case data := <-serialReceived:
		t.Logf("Serial port received: %v", data)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for serial port to receive data")
	}

	t.Log("Full bidirectional flow works!")
}

func TestPTYWithFramer(t *testing.T) {
	// Create PTY
	pty, err := OpenPTY()
	if err != nil {
		t.Fatalf("Failed to create PTY: %v", err)
	}
	defer pty.Close()

	// Use go.bug.st/serial to open slave
	mode := &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(pty.SlavePath, mode)
	if err != nil {
		t.Fatalf("Failed to open serial port: %v", err)
	}
	defer port.Close()

	if err := port.SetReadTimeout(100 * time.Millisecond); err != nil {
		t.Logf("Warning: failed to set read timeout: %v", err)
	}

	// Create framers for both sides
	// Import path for meshtastic package
	masterFramer := newStreamFramer(pty.Master, pty.Master)
	slaveFramer := newStreamFramer(port, port)

	// Start master read loop (simulating simulator)
	masterReceived := make(chan []byte, 10)
	go func() {
		for {
			pty.Master.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			data, err := masterFramer.ReadPacket()
			if err != nil {
				continue
			}
			masterReceived <- data
		}
	}()

	// Write from slave using framer
	testPayload := []byte{0x18, 0x01} // WantConfig
	err = slaveFramer.WritePacket(testPayload)
	if err != nil {
		t.Fatalf("Failed to write packet: %v", err)
	}
	t.Logf("Wrote packet via slave framer")

	// Wait for master to receive
	select {
	case data := <-masterReceived:
		t.Logf("Master received packet: %v", data)
		if len(data) != len(testPayload) {
			t.Errorf("Expected %d bytes, got %d", len(testPayload), len(data))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for master to receive packet")
	}

	t.Log("Framer test passed!")
}

// Simple StreamFramer implementation for testing (mirrors pkg/meshtastic/framing.go)
type testStreamFramer struct {
	reader interface{ Read([]byte) (int, error) }
	writer interface{ Write([]byte) (int, error) }
}

func newStreamFramer(r interface{ Read([]byte) (int, error) }, w interface{ Write([]byte) (int, error) }) *testStreamFramer {
	return &testStreamFramer{reader: r, writer: w}
}

func (f *testStreamFramer) ReadPacket() ([]byte, error) {
	// Read header (4 bytes)
	header := make([]byte, 4)
	total := 0
	for total < 4 {
		n, err := f.reader.Read(header[total:])
		if err != nil {
			return nil, err
		}
		total += n
	}

	// Validate magic
	if header[0] != 0x94 || header[1] != 0xc3 {
		return nil, os.ErrInvalid
	}

	// Get length
	length := int(header[2])<<8 | int(header[3])
	if length > 512 {
		return nil, os.ErrInvalid
	}

	// Read payload
	payload := make([]byte, length)
	total = 0
	for total < length {
		n, err := f.reader.Read(payload[total:])
		if err != nil {
			return nil, err
		}
		total += n
	}

	return payload, nil
}

func (f *testStreamFramer) WritePacket(data []byte) error {
	// Build full packet (header + data)
	packet := make([]byte, 4+len(data))
	packet[0] = 0x94
	packet[1] = 0xc3
	packet[2] = byte(len(data) >> 8)
	packet[3] = byte(len(data))
	copy(packet[4:], data)

	// Write in one call
	_, err := f.writer.Write(packet)
	return err
}

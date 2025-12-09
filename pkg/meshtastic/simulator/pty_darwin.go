//go:build darwin

package simulator

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// TIOCPTYGNAME is the ioctl to get the slave PTY name on macOS
const TIOCPTYGNAME = 0x40807453

// PTY represents a pseudo-terminal pair
type PTY struct {
	Master    *os.File
	Slave     *os.File
	SlavePath string
}

// OpenPTY creates a new pseudo-terminal pair
func OpenPTY() (*PTY, error) {
	// Open the master side
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open /dev/ptmx: %w", err)
	}

	// Grant access to the slave (no-op on modern macOS)
	if err := grantpt(master); err != nil {
		_ = master.Close()
		return nil, fmt.Errorf("grantpt failed: %w", err)
	}

	// Unlock the slave (no-op on modern macOS)
	if err := unlockpt(master); err != nil {
		_ = master.Close()
		return nil, fmt.Errorf("unlockpt failed: %w", err)
	}

	// Get the slave path
	slavePath, err := ptsname(master)
	if err != nil {
		_ = master.Close()
		return nil, fmt.Errorf("ptsname failed: %w", err)
	}

	// Note: We don't open the slave side here.
	// The slave path is returned so another process (like the serial library)
	// can open it. Only one side should open the slave to avoid conflicts.

	// Set raw mode on master to avoid terminal processing affecting data
	if err := setRawMode(int(master.Fd())); err != nil {
		_ = master.Close()
		return nil, fmt.Errorf("failed to set raw mode: %w", err)
	}

	return &PTY{
		Master:    master,
		Slave:     nil, // Slave not opened by simulator
		SlavePath: slavePath,
	}, nil
}

// setRawMode configures the terminal for raw binary I/O
func setRawMode(fd int) error {
	// On macOS, use TIOCGETA/TIOCSETA instead of TCGETS/TCSETS
	termios, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	if err != nil {
		return err
	}

	// Set raw mode flags
	termios.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	termios.Oflag &^= unix.OPOST
	termios.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	termios.Cflag &^= unix.CSIZE | unix.PARENB
	termios.Cflag |= unix.CS8

	// Set read timeout behavior (VMIN=1, VTIME=0 for blocking read with 1 byte minimum)
	termios.Cc[unix.VMIN] = 1
	termios.Cc[unix.VTIME] = 0

	return unix.IoctlSetTermios(fd, unix.TIOCSETA, termios)
}

// Close closes both ends of the PTY
func (p *PTY) Close() error {
	var err error
	if p.Slave != nil {
		if e := p.Slave.Close(); e != nil {
			err = e
		}
	}
	if p.Master != nil {
		if e := p.Master.Close(); e != nil {
			err = e
		}
	}
	return err
}

// grantpt grants access to the slave pseudo-terminal
// On modern macOS, this is typically a no-op as permissions are handled automatically.
func grantpt(_ *os.File) error {
	// Modern macOS handles permissions automatically
	return nil
}

// unlockpt unlocks the slave pseudo-terminal
// On modern macOS, PTYs are unlocked by default.
func unlockpt(_ *os.File) error {
	// Modern macOS doesn't require explicit unlock
	return nil
}

// ptsname returns the path of the slave pseudo-terminal
func ptsname(f *os.File) (string, error) {
	// On macOS, use TIOCPTYGNAME to get the slave name
	// The buffer size is 128 bytes (PATH_MAX for PTY names)
	var buf [128]byte
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), TIOCPTYGNAME, uintptr(unsafe.Pointer(&buf[0])))
	if errno != 0 {
		return "", errno
	}

	// Find the null terminator
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i]), nil
		}
	}
	return string(buf[:]), nil
}

// CreateSymlink creates a symlink to the slave device at the given path
func (p *PTY) CreateSymlink(path string) error {
	// Remove existing symlink if it exists
	_ = os.Remove(path)
	return os.Symlink(p.SlavePath, path)
}

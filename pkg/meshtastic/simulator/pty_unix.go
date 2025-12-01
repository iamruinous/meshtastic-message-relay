//go:build unix

package simulator

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

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

	// Grant access to the slave
	if err := grantpt(master); err != nil {
		_ = master.Close()
		return nil, fmt.Errorf("grantpt failed: %w", err)
	}

	// Unlock the slave
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
	termios, err := unix.IoctlGetTermios(fd, unix.TCGETS)
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

	return unix.IoctlSetTermios(fd, unix.TCSETS, termios)
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
// On modern Linux with devpts filesystem, this is a no-op as permissions
// are handled automatically by the kernel.
func grantpt(f *os.File) error {
	// Modern devpts doesn't require explicit grantpt - it's handled by the kernel
	// Just verify the fd is valid by getting the PTY number
	var ptyno uint32
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TIOCGPTN, uintptr(unsafe.Pointer(&ptyno)))
	if errno != 0 {
		return errno
	}
	return nil
}

// unlockpt unlocks the slave pseudo-terminal
func unlockpt(f *os.File) error {
	var unlock int32
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&unlock)))
	if errno != 0 {
		return errno
	}
	return nil
}

// ptsname returns the path of the slave pseudo-terminal
func ptsname(f *os.File) (string, error) {
	var ptyno uint32
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TIOCGPTN, uintptr(unsafe.Pointer(&ptyno)))
	if errno != 0 {
		return "", errno
	}
	return fmt.Sprintf("/dev/pts/%d", ptyno), nil
}

// CreateSymlink creates a symlink to the slave device at the given path
func (p *PTY) CreateSymlink(path string) error {
	// Remove existing symlink if it exists
	_ = os.Remove(path)
	return os.Symlink(p.SlavePath, path)
}

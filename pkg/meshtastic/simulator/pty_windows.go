//go:build windows

package simulator

import (
	"errors"
	"os"
)

// ErrNotSupported is returned when PTY operations are attempted on Windows
var ErrNotSupported = errors.New("PTY simulation is not supported on Windows; use virtual COM port software like com0com for serial port testing")

// PTY represents a pseudo-terminal pair
// On Windows, this is a stub that returns errors for all operations.
type PTY struct {
	Master    *os.File
	Slave     *os.File
	SlavePath string
}

// OpenPTY creates a new pseudo-terminal pair
// On Windows, this always returns ErrNotSupported.
func OpenPTY() (*PTY, error) {
	return nil, ErrNotSupported
}

// Close closes both ends of the PTY
func (p *PTY) Close() error {
	if p == nil {
		return nil
	}
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

// CreateSymlink creates a symlink to the slave device at the given path
// On Windows, this always returns ErrNotSupported.
func (p *PTY) CreateSymlink(_ string) error {
	return ErrNotSupported
}

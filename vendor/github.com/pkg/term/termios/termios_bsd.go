// +build darwin freebsd openbsd netbsd dragonfly

package termios

import (
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	FREAD  = 0x0001
	FWRITE = 0x0002

	IXON       = 0x00000200
	IXOFF      = 0x00000400
	IXANY      = 0x00000800
	CCTS_OFLOW = 0x00010000
	CRTS_IFLOW = 0x00020000
	CRTSCTS    = CCTS_OFLOW | CRTS_IFLOW
)

// Tcgetattr gets the current serial port settings.
func Tcgetattr(fd uintptr, argp *unix.Termios) error {
	return ioctl(fd, unix.TIOCGETA, uintptr(unsafe.Pointer(argp)))
}

// Tcsetattr sets the current serial port settings.
func Tcsetattr(fd, opt uintptr, argp *unix.Termios) error {
	switch opt {
	case TCSANOW:
		opt = unix.TIOCSETA
	case TCSADRAIN:
		opt = unix.TIOCSETAW
	case TCSAFLUSH:
		opt = unix.TIOCSETAF
	default:
		return unix.EINVAL
	}
	return ioctl(fd, opt, uintptr(unsafe.Pointer(argp)))
}

// Tcsendbreak function transmits a continuous stream of zero-valued bits for
// four-tenths of a second to the terminal referenced by fildes. The duration
// parameter is ignored in this implementation.
func Tcsendbreak(fd, duration uintptr) error {
	if err := ioctl(fd, unix.TIOCSBRK, 0); err != nil {
		return err
	}
	time.Sleep(4 / 10 * time.Second)
	return ioctl(fd, unix.TIOCCBRK, 0)
}

// Tcdrain waits until all output written to the terminal referenced by fd has been transmitted to the terminal.
func Tcdrain(fd uintptr) error {
	return ioctl(fd, unix.TIOCDRAIN, 0)
}

// Tcflush discards data written to the object referred to by fd but not transmitted, or data received but not read, depending on the value of which.
func Tcflush(fd, which uintptr) error {
	var com int
	switch which {
	case unix.TCIFLUSH:
		com = FREAD
	case unix.TCOFLUSH:
		com = FWRITE
	case unix.TCIOFLUSH:
		com = FREAD | FWRITE
	default:
		return unix.EINVAL
	}
	return ioctl(fd, unix.TIOCFLUSH, uintptr(unsafe.Pointer(&com)))
}

// Cfgetispeed returns the input baud rate stored in the termios structure.
func Cfgetispeed(attr *unix.Termios) uint32 { return uint32(attr.Ispeed) }

// Cfgetospeed returns the output baud rate stored in the termios structure.
func Cfgetospeed(attr *unix.Termios) uint32 { return uint32(attr.Ospeed) }

// Tiocinq returns the number of bytes in the input buffer.
func Tiocinq(fd uintptr, argp *int) error {
	*argp = 0
	return nil
}

// Tiocoutq return the number of bytes in the output buffer.
func Tiocoutq(fd uintptr, argp *int) error {
	return ioctl(fd, unix.TIOCOUTQ, uintptr(unsafe.Pointer(argp)))
}

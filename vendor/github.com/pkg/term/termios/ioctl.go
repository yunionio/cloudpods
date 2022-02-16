// +build !windows,!solaris

package termios

import "golang.org/x/sys/unix"

func ioctl(fd, request, argp uintptr) error {
	if _, _, e := unix.Syscall6(unix.SYS_IOCTL, fd, request, argp, 0, 0, 0); e != 0 {
		return e
	}
	return nil
}

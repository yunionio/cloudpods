// +build !windows

package client

import (
	"syscall"
)

func newWaitStatus(ws uint32) syscall.WaitStatus {
	return syscall.WaitStatus(ws)
}

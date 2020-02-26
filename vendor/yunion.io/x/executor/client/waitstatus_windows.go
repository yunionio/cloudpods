package client

import (
	"syscall"
)

func NewWaitStatus(ws uint32) syscall.WaitStatus {
	return syscall.WaitStatus{
		ExitCode: ws,
	}
}

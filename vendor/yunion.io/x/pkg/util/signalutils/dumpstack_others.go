// +build !windows

package signalutils

import (
	"syscall"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"
)

func SetDumpStackSignal() {
	RegisterSignal(func() {
		utils.DumpAllGoroutineStack(log.Logger().Out)
	}, syscall.SIGUSR1)
}

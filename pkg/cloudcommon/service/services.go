package service

import (
	"os"
	"syscall"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/signalutils"
	"yunion.io/x/pkg/utils"
)

type SServiceBase struct{}

func (s *SServiceBase) RegisterSignals(quitHandler signalutils.Trap) {
	quitSignals := []os.Signal{syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM}
	signalutils.RegisterSignal(quitHandler, quitSignals...)

	// dump goroutine stack
	signalutils.RegisterSignal(func() {
		utils.DumpAllGoroutineStack(log.Logger().Out)
	}, syscall.SIGUSR1)

	signalutils.StartTrap()
}

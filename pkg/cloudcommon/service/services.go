package service

import (
	"os"
	"syscall"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/signalutils"
	"yunion.io/x/pkg/utils"
)

type SServiceBase struct{}

func (s *SServiceBase) RegisterQuitSignals(quitHandler signalutils.Trap) {
	quitSignals := []os.Signal{syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM}
	signalutils.RegisterSignal(quitHandler, quitSignals...)

	signalutils.StartTrap()
}

func (s *SServiceBase) RegisterSIGUSR1() {
	// dump goroutine stack
	signalutils.RegisterSignal(func() {
		utils.DumpAllGoroutineStack(log.Logger().Out)
	}, syscall.SIGUSR1)
}

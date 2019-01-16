package service

import (
	"os"
	"syscall"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/signalutils"
	"yunion.io/x/pkg/utils"
)

type IServiceBase interface {
	StartService()
	ExitService()
	TrapSignals(signalutils.Trap)
}

type SServiceBase struct {
}

func (s *SServiceBase) TrapSignals(quitHandler signalutils.Trap) {
	quitSignals := []os.Signal{syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM}
	signalutils.RegisterSignal(quitHandler, quitSignals...)
	dumpStack := func() {
		utils.DumpAllGoroutineStack(log.Logger().Out)
	}
	signalutils.RegisterSignal(dumpStack, syscall.SIGUSR1)
	signalutils.StartTrap()
}

func (s *SServiceBase) StartService() {
	log.Infof("Base Start Service ...")
}

func (s *SServiceBase) ExitService() {
	log.Infof("Base Exit Service ...")
}

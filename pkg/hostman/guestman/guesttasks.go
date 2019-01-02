package guestman

import (
	"context"
	"time"

	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/util/timeutils2"
)

type SGuestStopTask struct {
	*SKVMGuestInstance
	ctx            context.Context
	timeout        int64
	startPowerdown *time.Time
}

func NewGuestStopTask(guest *SKVMGuestInstance, ctx context.Context, timeout int64) *SGuestStopTask {
	return &SGuestStopTask{
		SKVMGuestInstance: guest,
		ctx:               ctx,
		timeout:           timeout,
		startPowerdown:    nil,
	}
}

func (s *SGuestStopTask) Start() {
	if s.IsRunning() && s.IsMonitorAlive() {
		// Do Powerdown,
		s.monitor.SimpleCommand("system_powerdown", s.onPowerdownGuest)
	} else {
		s.CheckGuestRunningLater()
	}
}

func (s *SGuestStopTask) onPowerdownGuest(results string) {
	s.ExitCleanup(true)
	s.startPowerdown = &time.Now()
	s.checkGuestRunning()
}

func (s *SGuestStopTask) checkGuestRunning() {
	if !s.IsRunning() || time.Now().Sub(*s.startPowerdown) > (s.timeout*time.Duration) {
		s.Stop() // force stop
		hostutils.TaskComplete(s.ctx, nil)
	} else {
		s.CheckGuestRunningLater()
	}
}

func (s *SGuestStopTask) CheckGuestRunningLater() {
	timeutils2.AddTimeout(time.Second*1, s.checkGuestRunning())
}

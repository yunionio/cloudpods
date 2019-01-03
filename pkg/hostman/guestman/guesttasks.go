package guestman

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/util/timeutils2"
)

type IGuestTasks interface {
	Start(func(error))
}

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

type SGuestSyncConfigTaskExecutor struct {
	ctx   context.Context
	guest *SKVMGuestInstance
	tasks IGuestTasks

	errors   []error
	callback func([]error)
}

func NewGuestSyncConfigTaskExecutor(ctx context.Context, guest *SKVMGuestInstance, tasks []IGuestTasks, callback func(error)) *SGuestSyncConfigTaskExecutor {
	return &SGuestSyncConfigTaskExecutor{ctx, guest, tasks, make([]error, 0), callback}
}

func (t *SGuestSyncConfigTaskExecutor) Start(delay int) {
	timeutils2.AddTimeout(1*time.Second, t.runNextTask)
}

func (t *SGuestSyncConfigTaskExecutor) runNextTask() {
	if len(t.tasks) > 0 {
		task := t.tasks[len(t.tasks-1)]
		t.tasks = t.tasks[:len(t.tasks)-1]
		task.Start(t.runNextTaskCallback)
	} else {
		t.doCallback()
	}
}

func (t *SGuestSyncConfigTaskExecutor) doCallback() {
	if t.callback != nil {
		t.callback(t.errors)
		t.callback = nil
	}
}

func (t *SGuestSyncConfigTaskExecutor) runNextTaskCallback(err error) {
	if err != nil {
		t.errors = append(t.errors, err)
	}
	t.runNextTask()
}

type SGuestDiskSyncTask struct {
	guest    *SKVMGuestInstance
	delDisks []jsonutils.JSONObject
	addDisks []jsonutils.JSONObject
	cdrom    string

	callback func(error)
}

func (d *SGuestDiskSyncTask) NewGuestDiskSyncTask(guest *SKVMGuestInstance, delDisks, addDisks []jsonutils.JSONObject, cdrom string) *SGuestDiskSyncTask {
	return &SGuestDiskSyncTask{guest, delDisks, addDisks, cdrom}
}

func (d *SGuestDiskSyncTask) Start(callback func(error)) {
	d.callback = callback
	d.syncDisksConf()
}

func (d *SGuestDiskSyncTask) syncDisksConf() {
	if len(d.delDisks) > 0 {
		disk = d.delDisks[len(d.delDisks)-1]
		d.delDisks = d.delDisks[:len(d.delDisks)-1]
		d.removeDisk(disk)
		return
	}
	if len(d.addDisks) > 0 {
		disk = d.addDisks[len(d.addDisks)-1]
		d.addDisks = d.addDisks[:len(d.addDisks)-1]
		d.addDisk(disk)
		return
	}
	if len(d.cdrom) > 0 {
		d.changeCdrom()
		return
	}
	d.callback()
}

def change_cdrom(self):
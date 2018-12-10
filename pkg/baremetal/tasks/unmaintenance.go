package tasks

import (
	"context"

	"yunion.io/x/jsonutils"

	baremetalstatus "yunion.io/x/onecloud/pkg/baremetal/status"
	"yunion.io/x/onecloud/pkg/baremetal/types"
)

type SBaremetalUnmaintenanceTask struct {
	*SBaremetalTaskBase
}

func NewBaremetalUnmaintenanceTask(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) *SBaremetalUnmaintenanceTask {
	task := new(SBaremetalUnmaintenanceTask)
	baseTask := newBaremetalTaskBase(baremetal, taskId, data)
	task.SBaremetalTaskBase = baseTask
	var err error
	if jsonutils.QueryBoolean(task.data, "guest_running", false) {
		err = task.EnsurePowerShutdown(false)
		if err != nil {
			SetTaskFail(task, err)
			return task
		}
		err = task.EnsurePowerUp("disk")
		if err != nil {
			SetTaskFail(task, err)
			return task
		}
		task.Baremetal.SyncStatus(baremetalstatus.RUNNING, "")
		SetTaskComplete(task, nil)
		return task
	}
	task.SetStage(task.WaitForStop)
	err = task.EnsurePowerShutdown(true)
	if err != nil {
		SetTaskFail(task, err)
		return task
	}
	ExecuteTask(task, nil)
	return task
}

func (self *SBaremetalUnmaintenanceTask) WaitForStop(ctx context.Context, args interface{}) error {
	status, err := self.Baremetal.GetPowerStatus()
	if err != nil {
		return err
	}
	self.SetStage(self.OnStopComplete)
	if status == types.POWER_STATUS_OFF {
		ExecuteTask(self, nil)
	}
	return nil
}

func (self *SBaremetalUnmaintenanceTask) OnStopComplete(ctx context.Context, args interface{}) error {
	self.Baremetal.SyncStatus(baremetalstatus.READY, "")
	SetTaskComplete(self, nil)
	return nil
}

func (self *SBaremetalUnmaintenanceTask) GetName() string {
	return "BaremetalUnmaintenanceTask"
}

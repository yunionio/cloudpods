package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	baremetalstatus "yunion.io/x/onecloud/pkg/baremetal/status"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
)

type SBaremetalUnmaintenanceTask struct {
	*SBaremetalTaskBase
}

func NewBaremetalUnmaintenanceTask(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) (ITask, error) {
	task := new(SBaremetalUnmaintenanceTask)
	baseTask := newBaremetalTaskBase(baremetal, taskId, data)
	task.SBaremetalTaskBase = baseTask
	var err error
	if jsonutils.QueryBoolean(task.data, "guest_running", false) {
		err = task.EnsurePowerShutdown(false)
		if err != nil {
			return task, fmt.Errorf("EnsurePowerShutdown hard: %v", err)
		}
		err = task.EnsurePowerUp("disk")
		if err != nil {
			return task, fmt.Errorf("EnsurePowerUp disk: %v", err)
		}
		task.Baremetal.SyncStatus(baremetalstatus.RUNNING, "")
		SetTaskComplete(task, nil)
		return task, nil
	}
	task.SetStage(task.WaitForStop)
	err = task.EnsurePowerShutdown(true)
	if err != nil {
		return task, fmt.Errorf("EnsurePowerShutdown soft: %v", err)
	}
	ExecuteTask(task, nil)
	return task, nil
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

package tasks

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/baremetal/utils/ipmitool"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

type SBaremetalResetBMCTask struct {
	*SBaremetalPXEBootTaskBase
	term *ssh.Client
}

func NewBaremetalResetBMCTask(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) (ITask, error) {
	task := new(SBaremetalResetBMCTask)
	baseTask := newBaremetalPXEBootTaskBase(baremetal, taskId, data)
	task.SBaremetalPXEBootTaskBase = baseTask
	_, err := baseTask.InitPXEBootTask(task, data)
	return task, err
}

func (self *SBaremetalResetBMCTask) GetName() string {
	return "BaremetalResetBMCTask"
}

func (self *SBaremetalResetBMCTask) GetIPMITool() *ipmitool.SSHIPMI {
	return ipmitool.NewSSHIPMI(self.term)
}

func (self *SBaremetalResetBMCTask) OnPXEBoot(ctx context.Context, term *ssh.Client, args interface{}) error {
	self.term = term
	self.SetStage(self.WaitForBMCReady)
	err := ipmitool.DoBMCReset(self.GetIPMITool())
	if err != nil {
		return err
	}
	time.Sleep(10 * time.Second)
	ExecuteTask(self, nil)
	return nil
}

func (self *SBaremetalResetBMCTask) WaitForBMCReady(ctx context.Context, args interface{}) error {
	self.SetStage(self.OnBMCReady)
	status, err := ipmitool.GetChassisPowerStatus(self.GetIPMITool())
	if err != nil {
		return err
	}
	if status != "" && status == types.POWER_STATUS_ON {
		ExecuteTask(self, nil)
	}
	return nil
}

func (self *SBaremetalResetBMCTask) OnBMCReady(ctx context.Context, args interface{}) error {
	time.Sleep(20 * time.Second)
	SetTaskComplete(self, nil)
	return nil
}

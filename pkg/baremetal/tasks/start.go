package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
)

type SBaremetalServerStartTask struct {
	*SBaremetalTaskBase
}

func NewBaremetalServerStartTask(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) (*SBaremetalServerStartTask, error) {
	baseTask := newBaremetalTaskBase(baremetal, taskId, data)
	self := &SBaremetalServerStartTask{
		SBaremetalTaskBase: baseTask,
	}
	if err := self.Baremetal.DoDiskBoot(); err != nil {
		return nil, fmt.Errorf("DoDiskBoot: %v", err)
	}
	self.SetStage(self.WaitForStart)
	ExecuteTask(self, nil)
	return self, nil
}

func (self *SBaremetalServerStartTask) GetName() string {
	return "BaremetalServerStartTask"
}

func (self *SBaremetalServerStartTask) WaitForStart(ctx context.Context, args interface{}) error {
	self.SetStage(self.OnStartComplete)
	status, err := self.Baremetal.GetPowerStatus()
	if err != nil {
		return fmt.Errorf("GetPowerStatus: %v", err)
	}
	log.Infof("%s WaitForStart status=%s", self.GetName(), status)
	if status == types.POWER_STATUS_ON {
		ExecuteTask(self, nil)
	}
	return nil
}

func (self *SBaremetalServerStartTask) OnStartComplete(ctx context.Context, args interface{}) error {
	self.Baremetal.SyncAllStatus(types.POWER_STATUS_ON)
	SetTaskComplete(self, nil)
	return nil
}

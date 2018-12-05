package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
)

type SBaremetalServerBaseDeployTask struct {
	*SBaremetalPXEBootTaskBase
}

func newBaremetalServerBaseDeployTask(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
	queue *TaskQueue,
) *SBaremetalServerBaseDeployTask {
	task := new(SBaremetalServerBaseDeployTask)
	baseTask := newBaremetalPXEBootTaskBase(baremetal, taskId, data, task)
	task.SBaremetalPXEBootTaskBase = baseTask
	return task
}

func (self *SBaremetalServerBaseDeployTask) GetName() string {
	return "BaremetalServerBaseDeployTask"
}

func (self *SBaremetalServerBaseDeployTask) OnPXEBoot(ctx context.Context, args interface{}) error {
	log.Infof("%s called on stage pxeboot, args: %v", self.GetName(), args)
	return nil
}

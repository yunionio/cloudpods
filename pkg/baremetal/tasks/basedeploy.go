package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/util/ssh"
)

type IServerBaseDeployTask interface {
	IPXEBootTask
	DoDeploys(term *ssh.Client) (jsonutils.JSONObject, error)
}

type SBaremetalServerBaseDeployTask struct {
	*SBaremetalPXEBootTaskBase
	serverDeployTask IServerBaseDeployTask
}

func newBaremetalServerBaseDeployTask(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
	deployTask IServerBaseDeployTask,
) (*SBaremetalServerBaseDeployTask, error) {
	task := new(SBaremetalServerBaseDeployTask)
	baseTask := newBaremetalPXEBootTaskBase(baremetal, taskId, data)
	task.SBaremetalPXEBootTaskBase = baseTask
	_, err := baseTask.InitPXEBootTask(task, data)
	task.serverDeployTask = deployTask
	return task, err
}

func (self *SBaremetalServerBaseDeployTask) GetName() string {
	return "BaremetalServerBaseDeployTask"
}

func (self *SBaremetalServerBaseDeployTask) GetFinishAction() string {
	if self.data != nil {
		action, _ := self.data.GetString("on_finish")
		return action
	}
	return ""
}

func (self *SBaremetalServerBaseDeployTask) OnPXEBoot(ctx context.Context, term *ssh.Client, args interface{}) error {
	log.Infof("%s called on stage pxeboot, args: %v", self.GetName(), args)
	result, err := self.serverDeployTask.DoDeploys(term)
	if err != nil {
		return err
	}
	_, err = term.Run(
		"/bin/sync",
		"/sbin/sysctl -w vm.drop_caches=3",
	)
	if err != nil {
		return err
	}
	onFinishAction := self.GetFinishAction()
	if utils.IsInStringArray(onFinishAction, []string{"restart", "shutdown"}) {
		err = self.EnsurePowerShutdown(false)
		if err != nil {
			return err
		}
		if onFinishAction == "restart" {
			err = self.EnsurePowerUp("disk")
			if err != nil {
				return err
			}
		}
	}
	self.Baremetal.AutoSyncAllStatus()
	SetTaskComplete(self, result)
	return nil
}

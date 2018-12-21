package tasks

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/util/ssh"
)

type SBaremetalServerDeployTask struct {
	*SBaremetalServerBaseDeployTask
}

func NewBaremetalServerDeployTask(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) *SBaremetalServerDeployTask {
	task := new(SBaremetalServerDeployTask)
	baseTask := newBaremetalServerBaseDeployTask(baremetal, taskId, data, task)
	task.SBaremetalServerBaseDeployTask = baseTask
	return task
}

func (self *SBaremetalServerDeployTask) GetName() string {
	return "BaremetalServerDeployTask"
}

func (self *SBaremetalServerDeployTask) DoDeploys(term *ssh.Client) (jsonutils.JSONObject, error) {
	return self.Baremetal.GetServer().DoDeploy(term, self.data, false)
}

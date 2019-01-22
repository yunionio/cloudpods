package tasks

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/util/ssh"
)

type SBaremetalReprepareTask struct {
	*SBaremetalServerBaseDeployTask
}

func NewBaremetalReprepareTask(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) (ITask, error) {
	task := new(SBaremetalReprepareTask)
	baseTask, err := newBaremetalServerBaseDeployTask(baremetal, taskId, data, task)
	task.SBaremetalServerBaseDeployTask = baseTask
	return task, err
}

func (self *SBaremetalReprepareTask) GetName() string {
	return "BaremetalReprepareTask"
}

func (self *SBaremetalReprepareTask) DoDeploys(term *ssh.Client) (jsonutils.JSONObject, error) {
	task := newBaremetalPrepareTask(self.Baremetal)
	err := task.DoPrepare(term)
	return nil, err
}

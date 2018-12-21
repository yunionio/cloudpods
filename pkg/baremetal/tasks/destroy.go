package tasks

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/ssh"
)

type SBaremetalServerDestroyTask struct {
	*SBaremetalServerBaseDeployTask
}

func NewBaremetalServerDestroyTask(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) *SBaremetalServerDestroyTask {
	task := new(SBaremetalServerDestroyTask)
	baseTask := newBaremetalServerBaseDeployTask(baremetal, taskId, data, task)
	task.SBaremetalServerBaseDeployTask = baseTask
	return task
}

func (self *SBaremetalServerDestroyTask) GetName() string {
	return "BaremetalServerDestroyTask"
}

func (self *SBaremetalServerDestroyTask) DoDeploys(term *ssh.Client) (jsonutils.JSONObject, error) {
	if err := self.Baremetal.GetServer().DoEraseDisk(term); err != nil {
		log.Errorf("Delete server do erase disk: %v", err)
	}
	if err := self.Baremetal.GetServer().DoDiskUnconfig(term); err != nil {
		log.Errorf("Baremetal do disk unconfig: %v", err)
	}
	self.Baremetal.RemoveServer()
	return nil, nil
}

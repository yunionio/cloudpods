package tasks

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/util/ssh"
)

type SBaremetalServerRebuildTask struct {
	*SBaremetalServerBaseDeployTask
}

func NewBaremetalServerRebuildTask(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) (ITask, error) {
	task := new(SBaremetalServerRebuildTask)
	baseTask, err := newBaremetalServerBaseDeployTask(baremetal, taskId, data, task)
	task.SBaremetalServerBaseDeployTask = baseTask
	return task, err
}

func (self *SBaremetalServerRebuildTask) GetName() string {
	return "BaremetalServerRebuildTask"
}

func (self *SBaremetalServerRebuildTask) DoDeploys(term *ssh.Client) (jsonutils.JSONObject, error) {
	parts, err := self.Baremetal.GetServer().DoRebuildRootDisk(term)
	if err != nil {
		return nil, fmt.Errorf("Rebuild root disk: %v", err)
	}
	disks, err := self.Baremetal.GetServer().SyncPartitionSize(term, parts)
	if err != nil {
		return nil, fmt.Errorf("SyncPartitionSize: %v", err)
	}
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewArray(disks...), "disks")
	deployInfo, err := self.Baremetal.GetServer().DoDeploy(term, data, false)
	if err != nil {
		return nil, fmt.Errorf("DoDeploy: %v", err)
	}
	data.Update(deployInfo)
	return data, nil
}

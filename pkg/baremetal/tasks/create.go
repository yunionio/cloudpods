package tasks

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/ssh"
)

type SBaremetalServerCreateTask struct {
	*SBaremetalServerBaseDeployTask
}

func NewBaremetalServerCreateTask(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) *SBaremetalServerCreateTask {
	task := new(SBaremetalServerCreateTask)
	baseTask := newBaremetalServerBaseDeployTask(baremetal, taskId, data, task)
	task.SBaremetalServerBaseDeployTask = baseTask
	return task
}

func (self *SBaremetalServerCreateTask) GetName() string {
	return "BaremetalServerCreateTask"
}

func (self *SBaremetalServerCreateTask) DoDeploys(term *ssh.Client) (jsonutils.JSONObject, error) {
	// Build raid
	err := self.Baremetal.GetServer().DoDiskConfig(term)
	if err != nil {
		return nil, self.onError(term, err)
	}
	time.Sleep(2 * time.Second)
	if err := self.Baremetal.GetServer().DoEraseDisk(term); err != nil {
		return nil, self.onError(term, err)
	}
	time.Sleep(2 * time.Second)
	parts, err := self.Baremetal.GetServer().DoPartitionDisk(term)
	if err != nil {
		return nil, self.onError(term, err)
	}
	data := jsonutils.NewDict()
	disks, err := self.Baremetal.GetServer().SyncPartitionSize(term, parts)
	if err != nil {
		return nil, self.onError(term, err)
	}
	data.Add(jsonutils.Marshal(disks), "disks")
	deployInfo, err := self.Baremetal.GetServer().DoDeploy(term, data, true)
	if err != nil {
		return nil, self.onError(term, err)
	}
	if deployInfo != nil {
		data.Update(deployInfo)
	}
	return data, nil
}

func (self *SBaremetalServerCreateTask) onError(term *ssh.Client, err error) error {
	if err1 := self.Baremetal.GetServer().DoEraseDisk(term); err1 != nil {
		log.Warningf("EraseDisk error: %v", err1)
	}
	self.Baremetal.AutoSyncStatus()
	return err
}

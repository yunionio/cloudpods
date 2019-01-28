package tasks

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/util/ssh"
)

type SBaremetalMaintenanceTask struct {
	*SBaremetalPXEBootTaskBase
}

func NewBaremetalMaintenanceTask(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) (ITask, error) {
	task := new(SBaremetalMaintenanceTask)
	baseTask := newBaremetalPXEBootTaskBase(baremetal, taskId, data)
	task.SBaremetalPXEBootTaskBase = baseTask
	_, err := baseTask.InitPXEBootTask(task, data)
	return task, err
}

func (self *SBaremetalMaintenanceTask) OnPXEBoot(ctx context.Context, term *ssh.Client, args interface{}) error {
	sshConfig := term.GetConfig()
	dataObj := map[string]interface{}{
		"username": sshConfig.Username,
		"password": sshConfig.Password,
		"ip":       sshConfig.Host,
	}
	if jsonutils.QueryBoolean(self.data, "guest_running", false) {
		dataObj["guest_running"] = true
	}
	self.Baremetal.AutoSyncStatus()
	SetTaskComplete(self, jsonutils.Marshal(dataObj))
	return nil
}

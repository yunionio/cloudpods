package tasks

import (
	"context"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/baremetal/status"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

type SBaremetalServerPrepareTask struct {
	*SBaremetalTaskBase
}

func NewBaremetalServerPrepareTask(
	baremetal IBaremetal,
) *SBaremetalServerPrepareTask {
	baseTask := newBaremetalTaskBase(baremetal, "", nil)
	task := &SBaremetalServerPrepareTask{
		SBaremetalTaskBase: baseTask,
	}
	task.SetSSHStage(task.OnPXEBootRequest)
	return task
}

func (self *SBaremetalServerPrepareTask) NeedPXEBoot() bool {
	return true
}

func (self *SBaremetalServerPrepareTask) GetName() string {
	return "BaremetalServerPrepareTask"
}

// OnPXEBootRequest called by notify api handler
func (self *SBaremetalServerPrepareTask) OnPXEBootRequest(ctx context.Context, cli *ssh.Client, args interface{}) error {
	err := newBaremetalPrepareTask(self.Baremetal).DoPrepare(cli)
	if err != nil {
		log.Errorf("Prepare failed: %v", err)
		self.Baremetal.SyncStatus(status.PREPARE_FAIL, err.Error())
		return err
	}
	self.Baremetal.AutoSyncStatus()
	SetTaskComplete(self, nil)
	return nil
}

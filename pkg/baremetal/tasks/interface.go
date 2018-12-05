package tasks

import (
	"yunion.io/x/onecloud/pkg/baremetal/types"
)

type IBaremetal interface {
	GetTaskQueue() *TaskQueue
	GetSSHConfig() (*types.SSHConfig, error)
	TestSSHConfig() bool
	DoPowerShutdown(soft bool)
	GetAdminNic() *types.Nic
	GetName() string
}

package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var CronJobs *CronJobManager

type CronJobManager struct {
	*NamespaceResourceManager
}

func init() {
	CronJobs = &CronJobManager{
		NamespaceResourceManager: NewNamespaceResourceManager("cronjob", "cronjobs", NewNamespaceCols(), NewColumns()),
	}

	modules.Register(CronJobs)
}

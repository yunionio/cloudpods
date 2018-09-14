package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var Jobs *JobManager

type JobManager struct {
	*NamespaceResourceManager
}

func init() {
	Jobs = &JobManager{
		NamespaceResourceManager: NewNamespaceResourceManager("job", "jobs", NewNamespaceCols(), NewColumns()),
	}

	modules.Register(Jobs)
}

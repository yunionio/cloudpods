package k8s

import (
	"github.com/yunionio/onecloud/pkg/mcclient/modules"
)

var (
	Logs modules.ResourceManager
)

func init() {
	Logs = NewManager("kube_event", "kube_events",
		[]string{"id", "ops_time", "obj_id", "obj_type", "obj_name", "user", "user_id", "tenant", "tenant_id", "owner_tenant_id", "action", "notes"},
		[]string{})
	modules.Register(&Logs)
}

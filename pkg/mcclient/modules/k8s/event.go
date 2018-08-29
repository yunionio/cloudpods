package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Logs *ResourceManager
)

func init() {
	Logs = NewResourceManager("kube_event", "kube_events",
		NewColumns("id", "ops_time", "obj_id", "obj_type", "obj_name", "user", "user_id", "tenant", "tenant_id", "owner_tenant_id", "action", "notes"),
		NewColumns())
	modules.Register(Logs)
}

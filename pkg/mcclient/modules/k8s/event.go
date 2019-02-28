package k8s

var (
	Logs *ResourceManager
)

func init() {
	Logs = NewResourceManager("event", "events",
		NewColumns("id", "ops_time", "obj_id", "obj_type", "obj_name", "user", "user_id", "tenant", "tenant_id", "owner_tenant_id", "action", "notes"),
		NewColumns())
}

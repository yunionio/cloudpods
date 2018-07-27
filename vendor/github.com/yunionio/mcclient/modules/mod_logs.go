package modules

var (
	Logs ResourceManager
)

func init() {
	Logs = NewComputeManager("event", "events",
		[]string{"id", "ops_time", "obj_id", "obj_type", "obj_name", "user", "user_id", "tenant", "tenant_id", "owner_tenant_id", "action", "notes"},
		[]string{})
	registerCompute(&Logs)
}

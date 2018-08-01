package modules

var (
	Actions ResourceManager
)

func init() {
	Actions = NewActionManager("action", "actions",
		[]string{"id", "start_time", "ops_time", "obj_id", "obj_type", "obj_name", "user", "user_id", "tenant", "tenant_id", "owner_tenant_id", "action", "success", "notes"},
		[]string{})
	register(&Actions)
}

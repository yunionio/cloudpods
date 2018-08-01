package modules

var (
	MonitorTypes ResourceManager
)

func init() {
	MonitorTypes = NewMonitorManager("monitor_type", "monitor_types",
		[]string{"id", "name", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark"},
		[]string{})

	register(&MonitorTypes)
}

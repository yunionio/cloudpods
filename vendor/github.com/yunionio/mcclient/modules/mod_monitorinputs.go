package modules

var (
	MonitorInputs ResourceManager
)

func init() {
	MonitorInputs = NewMonitorManager("monitor_input", "monitor_inputs",
		[]string{"ID", "monitor_name", "monitor_parameters", "sample", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark"},
		[]string{})

	register(&MonitorInputs)
}

package modules

var (
	MonitorTemplates      ResourceManager
	MonitorTemplateInputs JointResourceManager
)

func init() {
	MonitorTemplates = NewMonitorManager("monitor_template", "monitor_templates",
		[]string{"ID", "monitor_template_name", "monitor_template_desc", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark"},
		[]string{})

	MonitorTemplateInputs = NewJointMonitorManager(
		"monitorInfo",
		"monitorInfos",
		[]string{"ID", "monitor_template_id", "monitor_name", "monitor_conf_value", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark"},
		[]string{},
		&MonitorTemplates,
		&MonitorInputs)

	register(&MonitorTemplates)

	register(&MonitorTemplateInputs)
}

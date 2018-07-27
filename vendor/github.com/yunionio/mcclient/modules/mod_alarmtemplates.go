package modules

var (
	AlarmTemplates ResourceManager
)

func init() {
	AlarmTemplates = NewMonitorManager("alarm_template", "alarm_templates",
		[]string{"id", "alarm_template_name", "alarm_template_desc", "belongto", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark"},
		[]string{})

	register(&AlarmTemplates)
}

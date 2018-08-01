package modules

var (
	AlarmTemplateAlarms JointResourceManager
)

func init() {
	AlarmTemplateAlarms = NewJointMonitorManager(
		"alarmtemplate_alarm",
		"alarmtemplate_alarms",
		[]string{"id", "alarm_template_name", "alarm_template_desc", "belongto", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark"},
		[]string{},
		&AlarmTemplates,
		&Alarms)

	register(&AlarmTemplateAlarms)
}

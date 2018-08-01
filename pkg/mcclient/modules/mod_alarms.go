package modules

var (
	Alarms ResourceManager
)

func init() {
	Alarms = NewMonitorManager("alarm", "alarms",
		[]string{"id", "metric_name", "unit", "common_unit", "alarm_condition", "template", "alarm_level", "contact_type", "expire_seconds", "escalate_seconds", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark"},
		[]string{})

	register(&Alarms)
}

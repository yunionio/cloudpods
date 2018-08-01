package modules

var (
	AlarmLogs ResourceManager
)

func init() {
	AlarmLogs = NewMonitorManager("alarm_log", "alarm_logs",
		[]string{"ID", "node_name", "metric_name", "labels", "start_time", "this_time", "alarm_ways", "alarm_level", "alarm_status", "receive_person", "alarm_release_time", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark"},
		[]string{})

	register(&AlarmLogs)
}

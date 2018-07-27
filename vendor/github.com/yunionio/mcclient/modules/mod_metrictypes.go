package modules

var (
	MetricsTypes ResourceManager
)

func init() {
	MetricsTypes = NewMonitorManager("metric_type", "metric_types",
		[]string{"id", "name", "description", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark"},
		[]string{})

	register(&MetricsTypes)
}

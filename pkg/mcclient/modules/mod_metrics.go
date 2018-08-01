package modules

var (
	Metrics ResourceManager
)

func init() {
	Metrics = NewMonitorManager("metric", "metrics",
		[]string{"id", "name", "description", "unit", "common_unit", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark"},
		[]string{})

	register(&Metrics)
}

package modules

var (
	NodeAlert ResourceManager
)

func init() {
	NodeAlert = NewMeterAlertManager("nodealert", "nodealerts",
		[]string{"id", "type", "metric", "node_name", "node_id", "period", "window", "comparator", "threshold", "recipients", "level", "channel", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark"},
		[]string{})

	register(&NodeAlert)
}

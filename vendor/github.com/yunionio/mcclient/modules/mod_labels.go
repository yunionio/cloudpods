package modules

var (
	Labels ResourceManager
)

func init() {
	Labels = NewMonitorManager("label", "labels",
		[]string{"ID", "label_name", "label_value", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark"},
		[]string{})

	register(&Labels)
}

package modules

var (
	ServiceTrees ResourceManager
)

func init() {
	ServiceTrees = NewMonitorManager("service_tree", "service_trees",
		[]string{"id", "service_tree_name", "service_tree_struct", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark"},
		[]string{})

	register(&ServiceTrees)
}

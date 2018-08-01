package modules

var (
	ProjectAdmin ResourceManager
)

func init() {
	ProjectAdmin = NewMonitorManager("project_admin", "project_admins",
		[]string{"create_by", "gmt_create", "id", "is_deleted", "node_labels", "officer_id", "officer_name", "type", "domain"},
		[]string{},
	)
	register(&ProjectAdmin)
}

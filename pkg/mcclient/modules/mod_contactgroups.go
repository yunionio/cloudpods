package modules

var (
	ContactGroups ResourceManager
)

func init() {
	ContactGroups = NewNotifyManager("contact-group", "contact-groups",
		[]string{"id", "name", "domain_id", "domain_name", "members", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "project_id", "remark"},
		[]string{})

	register(&ContactGroups)
}

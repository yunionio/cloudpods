package modules

var (
	Processes ResourceManager
)

func init() {
	Processes = NewITSMManager("process", "processes",
		[]string{"id", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark", "deploy_id", "deploy_category", "deploy_name", "deploy_time", "procdef_id", "procdef_name", "procdef_key", "procdef_version", "resource_name", "diagram_resource_name", "icon_name", "style_text", "shiro_name", "common_form", "start_form", "show_flag", "common_start_string"},
		[]string{"id", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark", "deploy_id", "deploy_category", "deploy_name", "deploy_time", "procdef_id", "procdef_name", "procdef_key", "procdef_version", "resource_name", "diagram_resource_name", "icon_name", "style_text", "shiro_name", "common_form", "start_form", "show_flag", "common_start_string"},
	)
	register(&Processes)
}

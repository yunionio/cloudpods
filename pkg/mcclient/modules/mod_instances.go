package modules

var (
	Instances ResourceManager
)

func init() {
	Instances = NewITSMManager("instance", "instances",
		[]string{"id", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark", "apply_date", "instance_id", "business_id", "procdef_key", "busi_type", "apply_unit", "client_info", "emergency", "impact", "title", "content", "start_time", "end_time", "starter", "starter_name", "instance_status", "current_approver", "approver_name", "task_name", "task_type", "task_id", "result", "login_name", "common_start_string"},
		[]string{"id", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark", "apply_date", "instance_id", "business_id", "procdef_key", "busi_type", "apply_unit", "client_info", "emergency", "impact", "title", "content", "start_time", "end_time", "starter", "starter_name", "instance_status", "current_approver", "approver_name", "task_name", "task_type", "task_id", "result", "login_name", "common_start_string"},
	)
	register(&Instances)
}

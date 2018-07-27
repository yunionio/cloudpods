package modules

var (
	Operations ResourceManager
)

func init() {
	Operations = NewITSMManager("operation", "operations",
		[]string{"id", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark", "instance_id", "business_id", "operate_type", "template_id", "device_code", "ip_address", "cpu", "memery", "disk", "resource_sum", "network", "schedule", "login_name", "display_name", "resource_ids", "instance", "task", "log_list", "common_start_string"},
		[]string{"id", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark", "instance_id", "business_id", "operate_type", "template_id", "device_code", "ip_address", "cpu", "memery", "disk", "resource_sum", "network", "schedule", "login_name", "display_name", "resource_ids", "instance", "task", "log_list", "common_start_string"},
	)
	register(&Operations)
}

package modules

var (
	Processlogs ResourceManager
)

func init() {
	Processlogs = NewITSMManager("processlog", "processlogs",
		[]string{"id", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark", "instance_id", "business_id", "receive_time", "task_receiver", "task_operator", "task_status", "task_name", "task_type", "task_id", "handle_time", "operate_result", "operate_advice", "log_order", "common_start_string"},
		[]string{"id", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark", "instance_id", "business_id", "receive_time", "task_receiver", "task_operator", "task_status", "task_name", "task_type", "task_id", "handle_time", "operate_result", "operate_advice", "log_order", "common_start_string"},
	)
	register(&Processlogs)
}

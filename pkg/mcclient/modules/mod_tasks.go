package modules

var (
	Tasks ResourceManager

	ComputeTasks ResourceManager
)

func init() {
	Tasks = NewITSMManager("task", "taskman",
		[]string{"id", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark", "instance_id", "business_id", "task_id", "task_type", "task_name", "task_status", "current_approver", "approver_name", "receive_time", "finish_time", "result", "content", "common_start_string"},
		[]string{"id", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark", "instance_id", "business_id", "task_id", "task_type", "task_name", "task_status", "current_approver", "approver_name", "receive_time", "finish_time", "result", "content", "common_start_string"},
	)
	register(&Tasks)

	ComputeTasks = NewComputeManager("task", "tasks",
		[]string{},
		[]string{"Id", "Obj_name", "Obj_Id", "Task_name", "Stage", "Created_at"})
	registerCompute(&ComputeTasks)
}

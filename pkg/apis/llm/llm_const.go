package llm

const (
	STATUS_READY = "ready"
)

const (
	/* 未知 */
	LLM_STATUS_UNKNOWN = "unknown"

	/* 创建失败 */
	LLM_STATUS_CREATE_FAIL = "create_fail"

	/* 启动失败 */
	LLM_STATUS_START_FAIL = "start_fail"
	/* 停机失败 */
	LLM_STATUS_STOP_FAILED = "stop_fail"

	/* 开始保存应用 */
	LLM_STATUS_START_SAVE_MODEL = "start_save_model"
	/* 正在保存应用 */
	LLM_STATUS_SAVING_MODEL = "saving_model"
	/* 保存应用失败 */
	LLM_STATUS_SAVE_MODEL_FAILED = "save_model_failed"

	/* 开始同步状态 */
	LLM_STATUS_START_SYNCSTATUS = "start_syncstatus"
	/* 正在同步状态 */
	LLM_STATUS_SYNCSTATUS = "syncstatus"

	/* 停机 */
	LLM_STATUS_READY = "ready"
	/* 运行 */
	LLM_STATUS_RUNNING = "running"

	/* 删除 */
	LLM_STATUS_DELETED = "deleted"

	LLM_LLM_STATUS_NO_SERVER    = "no_server"
	LLM_LLM_STATUS_NO_CONTAINER = "no_container"

	LLM_STATUS_START_DELETE  = "start_delete"
	LLM_STATUS_DELETING      = "deleting"
	LLM_STATUS_DELETE_FAILED = "delete_fail"
)

type TQuickModelMethod string

const (
	QuickModelInstall   = TQuickModelMethod("install")
	QuickModelUninstall = TQuickModelMethod("uninstall")
	QuickModelReinstall = TQuickModelMethod("reinstall")
)

const (
	LLM_PROBE_INSTANT_MODEl_INTERVAL_SECOND = 120 // 2 minute
)

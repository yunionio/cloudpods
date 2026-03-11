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

	/* 开始解绑磁盘 */
	LLM_STATUS_START_UNBIND = "start_unbind"
	/* 正在解绑磁盘 */
	LLM_STATUS_UNBINDING = "unbinding"
	/* 解绑磁盘失败 */
	LLM_STATUS_UNBIND_FAIL = "unbind_fail"

	/* 开始挂载磁盘 */
	LLM_STATUS_START_BIND = "start_bind"
	/* 正在挂载磁盘 */
	LLM_STATUS_BINDING = "binding"
	/* 挂载磁盘失败 */
	LLM_STATUS_BIND_FAIL = "bind_fail"

	/* 开始重启 */
	LLM_STATUS_START_RESTART = "start_restart"
	/* 正在重启 */
	LLM_STATUS_RESTARTING = "restarting"
	/* 重启失败 */
	LLM_STATUS_RESTART_FAILED = "restart_fail"

	/* 开始删除 */
	LLM_STATUS_START_DELETE = "start_delete"
	/* 正在删除 */
	LLM_STATUS_DELETING = "deleting"
	/* 删除失败 */
	LLM_STATUS_DELETE_FAILED = "delete_fail"

	/* 删除 */
	LLM_STATUS_DELETED = "deleted"

	LLM_LLM_STATUS_NO_SERVER    = "no_server"
	LLM_LLM_STATUS_NO_CONTAINER = "no_container"
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

type LLMEnvKey string

const (
	LLM_OPENCLAW_GATEWAY_TOKEN = LLMEnvKey("OPENCLAW_GATEWAY_TOKEN")
	LLM_OPENCLAW_AUTH_USERNAME = LLMEnvKey("AUTH_USERNAME")
	LLM_OPENCLAW_CUSTOM_USER   = LLMEnvKey("CUSTOM_USER")
	LLM_OPENCLAW_AUTH_PASSWORD = LLMEnvKey("AUTH_PASSWORD")
	LLM_OPENCLAW_PASSWORD      = LLMEnvKey("PASSWORD")
)

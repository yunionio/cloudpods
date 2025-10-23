package llm

const (
	STATUS_READY = "ready"
)

const (
	/* 未知 */
	LLM_STATUS_UNKOWN = "unkown"

	/* 创建失败 */
	LLM_STATUS_CREATE_FAIL = "create_fail"

	/* 启动失败 */
	LLM_STATUS_START_FAIL = "start_fail"
	/* 停机失败 */
	LLM_STATUS_STOP_FAILED = "stop_fail"

	/* 停机 */
	LLM_STATUS_READY = "ready"
	/* 运行 */
	LLM_STATUS_RUNNING = "running"

	LLM_STATUS_CREATING_POD             = "creating_pod"
	LLM_STATUS_CREAT_POD_FAILED         = "creat_pod_failed"
	LLM_STATUS_PULLING_MODEL            = "pulling_model"
	LLM_STATUS_GET_MANIFESTS_FAILED     = "get_manifests_failed"
	LLM_STATUS_DOWNLOADING_BLOBS        = "downloading_blobs"
	LLM_STATUS_DOWNLOADING_BLOBS_FAILED = "downloading_blobs_failed"
	LLM_STATUS_FETCHING_GGUF_FILE       = "fetching_gguf_file"
	LLM_STATUS_FETCH_GGUF_FILE_FAILED   = "fetch_gguf_failed"
	LLM_STATUS_CREATING_GGUF_MODEL      = "creating_gguf_model"
	LLM_STATUS_CREATE_GGUF_MODEL_FAILED = "create_gguf_model_failed"
	LLM_STATUS_PULLED_MODEL             = "pulled_model"
	LLM_STATUS_PULL_MODEL_FAILED        = "pull_model_failed"
	LLM_STATUS_START_DELETE             = "start_delete"
	LLM_STATUS_DELETING                 = "deleting"
	LLM_STATUS_DELETE_FAILED            = "delete_fail"
)

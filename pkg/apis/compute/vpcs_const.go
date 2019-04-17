package compute

const (
	VPC_STATUS_PENDING       = "pending"
	VPC_STATUS_AVAILABLE     = "available"
	VPC_STATUS_FAILED        = "failed"
	VPC_STATUS_START_DELETE  = "start_delete"
	VPC_STATUS_DELETING      = "deleting"
	VPC_STATUS_DELETE_FAILED = "delete_failed"
	VPC_STATUS_DELETED       = "deleted"
	VPC_STATUS_UNKNOWN       = "unknown"

	MAX_VPC_PER_REGION = 3

	DEFAULT_VPC_ID = "default"
	NORMAL_VPC_ID  = "normal" // 没有关联VPC的安全组，统一使用normal
)

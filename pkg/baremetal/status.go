package baremetal

const (
	INIT           = "init"
	PREPARE        = "prepare"
	PREPARE_FAIL   = "prepare_fail"
	READY          = "ready"
	RUNNING        = "running"
	MAINTAINING    = "maintaining"
	START_MAINTAIN = "start_maintain"
	DELETING       = "deleting"
	DELETE         = "delete"
	DELETE_FAIL    = "delete_fail"
	UNKNOWN        = "unknown"
	SYNCING_STATUS = "syncing_status"
	SYNC           = "sync"
	SYNC_FAIL      = "sync_fail"
	START_CONVERT  = "start_convert"
	CONVERTING     = "converting"
	START_FAIL     = "start_fail"
	STOP_FAIL      = "stop_fail"
)

const (
	NIC_TYPE_IPMI  = "ipmi"
	NIC_TYPE_ADMIN = "admin"
)

var NIC_TYPES = []string{NIC_TYPE_ADMIN, NIC_TYPE_IPMI}

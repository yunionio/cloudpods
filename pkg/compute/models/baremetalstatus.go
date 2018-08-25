package models

const (
	BAREMETAL_INIT           = "init"
	BAREMETAL_PREPARE        = "prepare"
	BAREMETAL_PREPARE_FAIL   = "prepare_fail"
	BAREMETAL_READY          = "ready"
	BAREMETAL_RUNNING        = "running"
	BAREMETAL_MAINTAINING    = "maintaining"
	BAREMETAL_START_MAINTAIN = "start_maintain"
	BAREMETAL_DELETING       = "deleting"
	BAREMETAL_DELETE         = "delete"
	BAREMETAL_DELETE_FAIL    = "delete_fail"
	BAREMETAL_UNKNOWN        = "unknown"
	BAREMETAL_SYNCING_STATUS = "syncing_status"
	BAREMETAL_SYNC           = "sync"
	BAREMETAL_SYNC_FAIL      = "sync_fail"
	BAREMETAL_START_CONVERT  = "start_convert"
	BAREMETAL_CONVERTING     = "converting"
	BAREMETAL_START_FAIL     = "start_fail"
	BAREMETAL_STOP_FAIL      = "stop_fail"
)

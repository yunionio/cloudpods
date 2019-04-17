package compute

const (
	// create by
	SNAPSHOT_MANUAL = "manual"
	SNAPSHOT_AUTO   = "auto"

	SNAPSHOT_CREATING    = "creating"
	SNAPSHOT_ROLLBACKING = "rollbacking"
	SNAPSHOT_FAILED      = "create_failed"
	SNAPSHOT_READY       = "ready"
	SNAPSHOT_DELETING    = "deleting"
	SNAPSHOT_UNKNOWN     = "unknown"
)

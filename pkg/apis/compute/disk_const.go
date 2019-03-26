package compute

const (
	DISK_INIT                = "init"
	DISK_REBUILD             = "rebuild"
	DISK_ALLOC_FAILED        = "alloc_failed"
	DISK_STARTALLOC          = "start_alloc"
	DISK_BACKUP_STARTALLOC   = "backup_start_alloc"
	DISK_BACKUP_ALLOC_FAILED = "backup_alloc_failed"
	DISK_ALLOCATING          = "allocating"
	DISK_READY               = "ready"
	DISK_RESET               = "reset"
	DISK_RESET_FAILED        = "reset_failed"
	DISK_DEALLOC             = "deallocating"
	DISK_DEALLOC_FAILED      = "dealloc_failed"
	DISK_UNKNOWN             = "unknown"
	DISK_DETACHING           = "detaching"
	DISK_ATTACHING           = "attaching"
	DISK_CLONING             = "cloning" // 硬盘克隆

	DISK_START_SAVE = "start_save"
	DISK_SAVING     = "saving"

	DISK_START_RESIZE = "start_resize"
	DISK_RESIZING     = "resizing"

	DISK_START_MIGRATE = "start_migrate"
	DISK_POST_MIGRATE  = "post_migrate"
	DISK_MIGRATING     = "migrating"

	DISK_START_SNAPSHOT = "start_snapshot"
	DISK_SNAPSHOTING    = "snapshoting"

	DISK_TYPE_SYS    = "sys"
	DISK_TYPE_SWAP   = "swap"
	DISK_TYPE_DATA   = "data"
	DISK_TYPE_VOLUME = "volume"

	DISK_BACKING_IMAGE = "image"
)

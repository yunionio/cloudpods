package consts

// TODO: move models/storages.go storage types to this file
const (
	STORAGE_LOCAL     = "local"
	STORAGE_BAREMETAL = "baremetal"
	STORAGE_SHEEPDOG  = "sheepdog"
	STORAGE_RBD       = "rbd"
	STORAGE_DOCKER    = "docker"
	STORAGE_NAS       = "nas"
	STORAGE_VSAN      = "vsan"
	STORAGE_NFS       = "nfs"

	DISK_TYPE_ROTATE = "rotate"
	DISK_TYPE_SSD    = "ssd"
)

var Local = []string{STORAGE_LOCAL, STORAGE_BAREMETAL, STORAGE_NAS}

const (
	STORAGE_ENABLED = "enabled"
	// STORAGE_DISABLED = "disabled"
	STORAGE_OFFLINE = "offline"
	STORAGE_ONLINE  = "online"
)

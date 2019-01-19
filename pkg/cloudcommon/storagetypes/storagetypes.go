package storagetypes

// TODO: move models/storages.go storage types to this file
var (
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

	Local = []string{STORAGE_LOCAL, STORAGE_BAREMETAL, STORAGE_NAS}
)

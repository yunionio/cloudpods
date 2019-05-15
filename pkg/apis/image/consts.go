package image

type TImageType string

const (
	// https://docs.openstack.org/glance/pike/user/statuses.html
	//
	IMAGE_STATUS_QUEUED     = "queued"
	IMAGE_STATUS_SAVING     = "saving"
	IMAGE_STATUS_ACTIVE     = "active"
	IMAGE_STATUS_CONVERTING = "converting"

	IMAGE_STATUS_DEACTIVATED    = "deactivated"
	IMAGE_STATUS_KILLED         = "killed"
	IMAGE_STATUS_DELETED        = "deleted"
	IMAGE_STATUS_PENDING_DELETE = "pending_delete"

	ImageTypeTemplate = TImageType("image")
	ImageTypeISO      = TImageType("iso")

	LocalFilePrefix = "file://"

	// image properties
	IMAGE_OS_ARCH             = "os_arch"
	IMAGE_OS_DISTRO           = "os_distribution"
	IMAGE_OS_TYPE             = "os_type"
	IMAGE_OS_VERSION          = "os_version"
	IMAGE_UEFI_SUPPORT        = "uefi_support"
	IMAGE_IS_LVM_PARTITION    = "is_lvm_partition"
	IMAGE_IS_READONLY         = "is_readonly"
	IMAGE_PARTITION_TYPE      = "partition_type"
	IMAGE_INSTALLED_CLOUDINIT = "installed_cloud_init"
)

var (
	ImageDeadStatus = []string{IMAGE_STATUS_DEACTIVATED, IMAGE_STATUS_KILLED, IMAGE_STATUS_DELETED, IMAGE_STATUS_PENDING_DELETE}
)

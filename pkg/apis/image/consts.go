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
)

var (
	ImageDeadStatus = []string{IMAGE_STATUS_DEACTIVATED, IMAGE_STATUS_KILLED, IMAGE_STATUS_DELETED, IMAGE_STATUS_PENDING_DELETE}
)

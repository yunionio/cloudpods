package huawei

import (
	"context"
	"strings"
	"time"

	"strconv"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type Attachment struct {
	ServerID     string `json:"server_id"`
	AttachmentID string `json:"attachment_id"`
	AttachedAt   string `json:"attached_at"`
	HostName     string `json:"host_name"`
	VolumeID     string `json:"volume_id"`
	Device       string `json:"device"`
	ID           string `json:"id"`
}

type DiskMeta struct {
	ResourceSpecCode string `json:"resourceSpecCode"`
	Billing          string `json:"billing"`
	ResourceType     string `json:"resourceType"`
	AttachedMode     string `json:"attached_mode"`
	Readonly         string `json:"readonly"`
}

type VolumeImageMetadata struct {
	QuickStart             string `json:"__quick_start"`
	ContainerFormat        string `json:"container_format"`
	MinRAM                 string `json:"min_ram"`
	ImageName              string `json:"image_name"`
	ImageID                string `json:"image_id"`
	OSType                 string `json:"__os_type"`
	OSFeatureList          string `json:"__os_feature_list"`
	MinDisk                string `json:"min_disk"`
	SupportKVM             string `json:"__support_kvm"`
	VirtualEnvType         string `json:"virtual_env_type"`
	Size                   string `json:"size"`
	OSVersion              string `json:"__os_version"`
	OSBit                  string `json:"__os_bit"`
	SupportKVMHi1822Hiovs  string `json:"__support_kvm_hi1822_hiovs"`
	SupportXen             string `json:"__support_xen"`
	Description            string `json:"__description"`
	Imagetype              string `json:"__imagetype"`
	DiskFormat             string `json:"disk_format"`
	ImageSourceType        string `json:"__image_source_type"`
	Checksum               string `json:"checksum"`
	Isregistered           string `json:"__isregistered"`
	HwVifMultiqueueEnabled string `json:"hw_vif_multiqueue_enabled"`
	Platform               string `json:"__platform"`
}

// https://support.huaweicloud.com/api-evs/zh-cn_topic_0124881427.html
type SDisk struct {
	storage *SStorage

	ID                  string              `json:"id"`
	Name                string              `json:"name"`
	Status              string              `json:"status"`
	Attachments         []Attachment        `json:"attachments"`
	Description         string              `json:"description"`
	Size                int64               `json:"size"`
	Metadata            DiskMeta            `json:"metadata"`
	Encrypted           bool                `json:"encrypted"`
	Bootable            string              `json:"bootable"`
	Multiattach         bool                `json:"multiattach"`
	AvailabilityZone    string              `json:"availability_zone"`
	SourceVolid         string              `json:"source_volid"`
	SnapshotID          string              `json:"snapshot_id"`
	CreatedAt           string              `json:"created_at"`
	VolumeType          string              `json:"volume_type"`
	VolumeImageMetadata VolumeImageMetadata `json:"volume_image_metadata"`
	ReplicationStatus   string              `json:"replication_status"`
	UserID              string              `json:"user_id"`
	ConsistencygroupID  string              `json:"consistencygroup_id"`
	UpdatedAt           string              `json:"updated_at"`

	/*下面这些字段也许不需要*/
	ExpiredTime time.Time
}

func (self *SDisk) GetId() string {
	return self.ID
}

func (self *SDisk) GetName() string {
	if len(self.Name) == 0 {
		return self.ID
	}

	return self.Name
}

func (self *SDisk) GetGlobalId() string {
	return self.ID
}

func (self *SDisk) GetStatus() string {
	// https://support.huaweicloud.com/api-evs/zh-cn_topic_0051803385.html
	switch self.Status {
	case "creating", "downloading":
		return models.DISK_ALLOCATING
	case "available", "in-use":
		return models.DISK_READY
	case "error":
		return models.DISK_ALLOC_FAILED
	case "attaching":
		return models.DISK_ATTACHING
	case "detaching":
		return models.DISK_DETACHING
	case "restoring-backup":
		return models.DISK_REBUILD
	case "backing-up":
		return models.DISK_BACKUP_STARTALLOC // ?
	case "error_restoring":
		return models.DISK_BACKUP_ALLOC_FAILED
	case "uploading":
		return models.DISK_SAVING //?
	case "extending":
		return models.DISK_RESIZING
	case "error_extending":
		return models.DISK_ALLOC_FAILED // ?
	case "deleting":
		return models.DISK_DEALLOC //?
	case "error_deleting":
		return models.DISK_DEALLOC_FAILED // ?
	case "rollbacking":
		return models.DISK_REBUILD
	case "error_rollbacking":
		return models.DISK_UNKNOWN
	default:
		return models.DISK_UNKNOWN
	}
}

func (self *SDisk) Refresh() error {
	new, err := self.storage.zone.region.GetDisk(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SDisk) IsEmulated() bool {
	return false
}

func (self *SDisk) GetMetadata() *jsonutils.JSONDict {
	// todo: add price key
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(models.HYPERVISOR_HUAWEI), "hypervisor")

	return data
}

func (self *SDisk) GetBillingType() string {
	// https://support.huaweicloud.com/api-evs/zh-cn_topic_0020235170.html
	if self.Metadata.Billing == "1" {
		return models.BILLING_TYPE_POSTPAID
	} else {
		return models.BILLING_TYPE_PREPAID // ?
	}
}

func (self *SDisk) GetExpiredAt() time.Time {
	return self.ExpiredTime
}

func (self *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return self.storage, nil
}

func (self *SDisk) GetDiskFormat() string {
	// self.volume_type ?
	return "vhd"
}

func (self *SDisk) GetDiskSizeMB() int {
	return int(self.Size * 1024)
}

func (self *SDisk) checkAutoDelete(attachments []Attachment) bool {
	autodelete := false
	for _, attach := range attachments {
		if len(attach.ServerID) > 0 {
			// todo : 忽略错误？？
			vm, err := self.storage.zone.region.GetInstanceByID(attach.ServerID)
			if err != nil {
				volumes := vm.OSExtendedVolumesVolumesAttached
				for _, vol := range volumes {
					if vol.ID == self.ID && strings.ToLower(vol.DeleteOnTermination) == "true" {
						autodelete = true
					}
				}
			}

			break
		}
	}

	return autodelete
}

func (self *SDisk) GetIsAutoDelete() bool {
	if len(self.Attachments) > 0 {
		return self.checkAutoDelete(self.Attachments)
	}

	return false
}

func (self *SDisk) GetTemplateId() string {
	return self.VolumeImageMetadata.ImageID
}

func (self *SDisk) GetDiskType() string {
	if self.Bootable == "true" {
		return models.DISK_TYPE_SYS
	} else {
		return models.DISK_TYPE_DATA
	}
}

func (self *SDisk) GetFsFormat() string {
	return ""
}

func (self *SDisk) GetIsNonPersistent() bool {
	return false
}

func (self *SDisk) GetDriver() string {
	// https://support.huaweicloud.com/api-evs/zh-cn_topic_0058762431.html
	// scsi or vbd?
	// todo: implement me
	return "scsi"
}

func (self *SDisk) GetCacheMode() string {
	return "none"
}

func (self *SDisk) GetMountpoint() string {
	if len(self.Attachments) > 0 {
		return self.Attachments[0].Device
	}

	return ""
}

func (self *SDisk) GetAccessPath() string {
	return ""
}

func (self *SDisk) Delete(ctx context.Context) error {
	panic("implement me")
}

func (self *SDisk) CreateISnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudSnapshot, error) {
	panic("implement me")
}

func (self *SDisk) getSnapshot(snapshotId string) (*SSnapshot, error) {
	snapshot, err := self.storage.zone.region.GetSnapshotById(snapshotId)
	return &snapshot, err
}

func (self *SDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	snapshot, err := self.getSnapshot(snapshotId)
	return snapshot, err
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots := make([]SSnapshot, 0)
	limit := 20
	offset := 0
	for {
		if parts, count, err := self.storage.zone.region.GetSnapshots(self.ID, "", offset, limit); err != nil {
			log.Errorf("GetDisks fail %s", err)
			return nil, err
		} else {
			snapshots = append(snapshots, parts...)
			if count < limit {
				break
			}

			offset += limit
		}
	}

	isnapshots := make([]cloudprovider.ICloudSnapshot, len(snapshots))
	for i := 0; i < len(snapshots); i++ {
		isnapshots[i] = &snapshots[i]
	}
	return isnapshots, nil
}

func (self *SDisk) Resize(ctx context.Context, newSizeMB int64) error {
	panic("implement me")
}

func (self *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	panic("implement me")
}

func (self *SDisk) Rebuild(ctx context.Context) error {
	panic("implement me")
}

func (self *SRegion) GetDisk(diskId string) (*SDisk, error) {
	var disk SDisk
	err := DoGet(self.ecsClient.Disks.Get, diskId, nil, &disk)
	return &disk, err
}

func (self *SRegion) GetDisks(zoneId string, offset int, limit int) ([]SDisk, int, error) {
	querys := map[string]string{}
	if len(zoneId) > 0 {
		querys["availability_zone"] = zoneId
	}

	querys["limit"] = strconv.Itoa(limit)
	querys["offset"] = strconv.Itoa(offset)

	disks := make([]SDisk, 0)
	err := DoList(self.ecsClient.Disks.List, querys, &disks)
	return disks, len(disks), err
}

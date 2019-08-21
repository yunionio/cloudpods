package openstack

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

const (
	DISK_STATUS_CREATING = "creating" // The volume is being created.

	DISK_STATUS_ATTACHING = "attaching" // The volume is attaching to an instance.
	DISK_STATUS_DETACHING = "detaching" // The volume is detaching from an instance.
	DISK_STATUS_EXTENDING = "extending" // The volume is being extended.
	DISK_STATUS_DELETING  = "deleting"  // The volume is being deleted.

	DISK_STATUS_RETYPING          = "retyping"          // The volume is changing type to another volume type.
	DISK_STATUS_AVAILABLE         = "available"         // The volume is ready to attach to an instance.
	DISK_STATUS_RESERVED          = "reserved"          // The volume is reserved for attaching or shelved.
	DISK_STATUS_IN_USE            = "in-use"            // The volume is attached to an instance.
	DISK_STATUS_MAINTENANCE       = "maintenance"       // The volume is locked and being migrated.
	DISK_STATUS_AWAITING_TRANSFER = "awaiting-transfer" // The volume is awaiting for transfer.
	DISK_STATUS_BACKING_UP        = "backing-up"        // The volume is being backed up.
	DISK_STATUS_RESTORING_BACKUP  = "restoring-backup"  // A backup is being restored to the volume.
	DISK_STATUS_DOWNLOADING       = "downloading"       // The volume is downloading an image.
	DISK_STATUS_UPLOADING         = "uploading"         // The volume is being uploaded to an image.

	DISK_STATUS_ERROR            = "error"            // A volume creation error occurred.
	DISK_STATUS_ERROR_DELETING   = "error_deleting"   // A volume deletion error occurred.
	DISK_STATUS_ERROR_BACKING_UP = "error_backing-up" // A backup error occurred.
	DISK_STATUS_ERROR_RESTORING  = "error_restoring"  // A backup restoration error occurred.
	DISK_STATUS_ERROR_EXTENDING  = "error_extending"  // An error occurred while attempting to extend a volume.

)

type Attachment struct {
	ServerID     string
	AttachmentID string
	HostName     string
	VolumeID     string
	Device       string
	ID           string
}

type Link struct {
	Href string
	Rel  string
}

type Metadata map[string]string

type VolumeImageMetadata struct {
	Checksum        string
	MinRAM          int
	DiskFormat      string
	ImageName       string
	ImageID         string
	ContainerFormat string
	MinDisk         int
	Size            int
}

type SDisk struct {
	storage *SStorage

	ID   string
	Name string

	MigrationStatus string
	Attachments     []Attachment
	Links           []Link

	AvailabilityZone  string
	Host              string `json:"os-vol-host-attr:host"`
	Encrypted         bool
	ReplicationStatus string
	SnapshotID        string
	Size              int
	UserID            string
	TenantID          string `json:"os-vol-tenant-attr:tenant_id"`
	Migstat           string `json:"os-vol-mig-status-attr:migstat"`
	Metadata          Metadata

	Status              string
	Description         string
	Multiattach         string
	SourceVolid         string
	ConsistencygroupID  string
	VolumeImageMetadata VolumeImageMetadata
	NameID              string `json:"os-vol-mig-status-attr:name_id"`
	Bootable            bool
	CreatedAt           time.Time
	VolumeType          string
}

func (disk *SDisk) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()

	data.Add(jsonutils.NewString(api.HYPERVISOR_OPENSTACK), "hypervisor")
	return data
}

func (region *SRegion) GetDisks(category, volumeBackendName string) ([]SDisk, error) {
	url := "/volumes/detail"
	disks := []SDisk{}
	for len(url) > 0 {
		_, resp, err := region.CinderList(url, "", nil)
		if err != nil {
			return nil, err
		}
		_disks := []SDisk{}
		err = resp.Unmarshal(&_disks, "volumes")
		if err != nil {
			return nil, errors.Wrap(err, `resp.Unmarshal(&_disks, "volumes")`)
		}
		disks = append(disks, _disks...)
		url = ""
		if resp.Contains("volumes_links") {
			nextLink := []SNextLink{}
			err = resp.Unmarshal(&nextLink, "volumes_links")
			if err != nil {
				return nil, errors.Wrap(err, `resp.Unmarshal(&nextLink, "volumes_links")`)
			}
			for _, next := range nextLink {
				if next.Rel == "next" {
					url = next.Href
					break
				}
			}
		}
	}
	result := []SDisk{}
	for _, disk := range disks {
		if len(category) == 0 || disk.VolumeType == category || strings.HasSuffix(disk.Host, "#"+volumeBackendName) {
			result = append(result, disk)
		}
	}
	return result, nil
}

func (disk *SDisk) GetId() string {
	return disk.ID
}

func (disk *SDisk) Delete(ctx context.Context) error {
	err := disk.storage.zone.region.DeleteDisk(disk.ID)
	if err != nil {
		return err
	}
	return cloudprovider.WaitDeleted(disk, 10*time.Second, 8*time.Minute)
}

func (disk *SDisk) attachInstances(attachments []Attachment) error {
	for _, attachment := range attachments {
		startTime := time.Now()
		for time.Now().Sub(startTime) < 5*time.Minute {
			if err := disk.storage.zone.region.AttachDisk(attachment.ServerID, disk.ID); err != nil {
				if strings.Contains(err.Error(), "status must be available or downloading") {
					time.Sleep(time.Second * 10)
					continue
				}
				log.Errorf("recover attach disk %s => instance %s error: %v", disk.ID, attachment.ServerID, err)
				return err
			} else {
				return nil
			}
		}
	}
	return nil
}

func (disk *SDisk) Resize(ctx context.Context, sizeMb int64) error {
	instanceIds := []string{}

	for _, attachement := range disk.Attachments {
		if err := disk.storage.zone.region.DetachDisk(attachement.ServerID, disk.ID); err != nil {
			return err
		}
		instanceIds = append(instanceIds, attachement.ServerID)
	}
	err := disk.storage.zone.region.ResizeDisk(disk.ID, sizeMb)
	if err != nil {
		disk.attachInstances(disk.Attachments)
		return err
	}
	return disk.attachInstances(disk.Attachments)
}

func (disk *SDisk) GetName() string {
	if len(disk.Name) > 0 {
		return disk.Name
	}
	return disk.ID
}

func (disk *SDisk) GetGlobalId() string {
	return disk.ID
}

func (disk *SDisk) IsEmulated() bool {
	return false
}

func (disk *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return disk.storage, nil
}

func (disk *SDisk) GetStatus() string {
	switch disk.Status {
	case DISK_STATUS_CREATING, DISK_STATUS_DOWNLOADING:
		return api.DISK_ALLOCATING
	case DISK_STATUS_ATTACHING:
		return api.DISK_ATTACHING
	case DISK_STATUS_DETACHING:
		return api.DISK_DETACHING
	case DISK_STATUS_EXTENDING:
		return api.DISK_RESIZING
	case DISK_STATUS_RETYPING, DISK_STATUS_AVAILABLE, DISK_STATUS_RESERVED, DISK_STATUS_IN_USE, DISK_STATUS_MAINTENANCE, DISK_STATUS_AWAITING_TRANSFER, DISK_STATUS_BACKING_UP, DISK_STATUS_RESTORING_BACKUP, DISK_STATUS_UPLOADING:
		return api.DISK_READY
	case DISK_STATUS_DELETING:
		return api.DISK_DEALLOC
	case DISK_STATUS_ERROR:
		return api.DISK_ALLOC_FAILED
	default:
		return api.DISK_UNKNOWN
	}
}

func (disk *SDisk) Refresh() error {
	new, err := disk.storage.zone.region.GetDisk(disk.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(disk, new)
}

func (disk *SDisk) ResizeDisk(sizeMb int64) error {
	return disk.storage.zone.region.ResizeDisk(disk.ID, sizeMb)
}

func (disk *SDisk) GetDiskFormat() string {
	return "lvm"
}

func (disk *SDisk) GetDiskSizeMB() int {
	return disk.Size * 1024
}

func (disk *SDisk) GetIsAutoDelete() bool {
	return false
}

func (disk *SDisk) GetTemplateId() string {
	return disk.VolumeImageMetadata.ImageID
}

func (disk *SDisk) GetDiskType() string {
	if disk.Bootable {
		return api.DISK_TYPE_SYS
	}
	return api.DISK_TYPE_DATA
}

func (disk *SDisk) GetFsFormat() string {
	return ""
}

func (disk *SDisk) GetIsNonPersistent() bool {
	return false
}

func (disk *SDisk) GetDriver() string {
	return "scsi"
}

func (disk *SDisk) GetCacheMode() string {
	return "none"
}

func (disk *SDisk) GetMountpoint() string {
	return ""
}

func (region *SRegion) CreateDisk(imageRef string, category string, name string, sizeGb int, desc string) (*SDisk, error) {
	params := map[string]map[string]interface{}{
		"volume": {
			"size":        sizeGb,
			"volume_type": category,
			"name":        name,
			"description": desc,
		},
	}
	if len(imageRef) > 0 {
		params["volume"]["imageRef"] = imageRef
	}
	_, resp, err := region.CinderCreate("/volumes", "", jsonutils.Marshal(params))
	if err != nil {
		return nil, err
	}

	disk := &SDisk{}
	if err := resp.Unmarshal(disk, "volume"); err != nil {
		return nil, err
	}
	//这里由于不好初始化disk的storage就手动循环了,如果是通过镜像创建，有个下载过程,比较慢，等待时间较长
	startTime := time.Now()
	timeout := time.Minute * 10
	//若是通过镜像创建，需要先下载镜像，需要的时间更长
	if len(imageRef) > 0 {
		timeout = time.Minute * 30
	}
	for time.Now().Sub(startTime) < timeout {
		disk, err = region.GetDisk(disk.GetGlobalId())
		if err != nil {
			return nil, err
		}
		log.Debugf("disk status %s expect %s", disk.GetStatus(), api.DISK_READY)
		status := disk.GetStatus()
		if status == api.DISK_READY {
			break
		}
		if status == api.DISK_ALLOC_FAILED {
			region.DeleteDisk(disk.GetGlobalId())
			return nil, fmt.Errorf("allocate disk failed, status is error")
		}
		time.Sleep(time.Second * 10)
	}
	if disk.GetStatus() != api.DISK_READY {
		region.DeleteDisk(disk.GetGlobalId())
		return nil, fmt.Errorf("timeout for waitting disk ready, current status: %s", disk.Status)
	}
	return disk, nil
}

func (region *SRegion) GetDisk(diskId string) (*SDisk, error) {
	_, resp, err := region.CinderGet("/volumes/"+diskId, "", nil)
	if err != nil {
		return nil, err
	}
	disk := &SDisk{}
	return disk, resp.Unmarshal(disk, "volume")
}

func (region *SRegion) DeleteDisk(diskId string) error {
	_, err := region.CinderDelete("/volumes/"+diskId, "")
	return err
}

func (region *SRegion) ResizeDisk(diskId string, sizeMb int64) error {
	params := map[string]map[string]interface{}{
		"os-extend": {
			"new_size": sizeMb / 1024,
		},
	}
	_, _, err := region.CinderAction(fmt.Sprintf("/volumes/%s/action", diskId), "", jsonutils.Marshal(params))
	return err
}

func (region *SRegion) ResetDisk(diskId, snapshotId string) error {
	//目前测试接口不能使用
	return cloudprovider.ErrNotSupported
	// params := map[string]map[string]interface{}{
	// 	"revert": {
	// 		"snapshot_id": snapshotId,
	// 	},
	// }
	// _, _, err := region.CinderAction(fmt.Sprintf("/volumes/%s/action", diskId), "3.40", jsonutils.Marshal(params))
	// return err
}

func (disk *SDisk) CreateISnapshot(ctx context.Context, name, desc string) (cloudprovider.ICloudSnapshot, error) {
	snapshot, err := disk.storage.zone.region.CreateSnapshot(disk.ID, name, desc)
	if err != nil {
		return nil, err
	}
	return snapshot, cloudprovider.WaitStatus(snapshot, api.SNAPSHOT_READY, time.Second*5, time.Minute*5)
}

func (disk *SDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return disk.storage.zone.region.GetISnapshotById(snapshotId)
}

func (disk *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return disk.storage.zone.region.GetSnapshots(disk.ID)
}

func (disk *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return disk.ID, disk.storage.zone.region.ResetDisk(disk.ID, snapshotId)
}

func (disk *SDisk) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (disk *SDisk) GetCreatedAt() time.Time {
	return disk.CreatedAt
}

func (disk *SDisk) GetExpiredAt() time.Time {
	return time.Time{}
}

func (disk *SDisk) GetAccessPath() string {
	return ""
}

func (disk *SDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (disk *SDisk) GetProjectId() string {
	return disk.TenantID
}

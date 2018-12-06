package azure

import (
	"fmt"
	"strings"
	"time"

	"context"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type DiskSku struct {
	Name string `json:"name,omitempty"`
	Tier string `json:"tier,omitempty"`
}

type ImageDiskReference struct {
	ID  string
	Lun int32 `json:"lun,omitempty"`
}

type CreationData struct {
	CreateOption     string `json:"createOption,omitempty"`
	StorageAccountID string
	ImageReference   *ImageDiskReference `json:"imageReference,omitempty"`
	SourceURI        string              `json:"sourceUri,omitempty"`
	SourceResourceID string              `json:"sourceResourceId,omitempty"`
}

type DiskProperties struct {
	//TimeCreated       time.Time //??? 序列化出错？
	OsType            string       `json:"osType,omitempty"`
	CreationData      CreationData `json:"creationData,omitempty"`
	DiskSizeGB        int32        `json:"diskSizeGB,omitempty"`
	ProvisioningState string       `json:"provisioningState,omitempty"`
	DiskState         string       `json:"diskState,omitempty"`
}

type SDisk struct {
	storage *SStorage

	ManagedBy  string         `json:"managedBy,omitempty"`
	Sku        DiskSku        `json:"sku,omitempty"`
	Zones      []string       `json:"zones,omitempty"`
	ID         string         `json:"id,omitempty"`
	Name       string         `json:"name,omitempty"`
	Type       string         `json:"type,omitempty"`
	Location   string         `json:"location,omitempty"`
	Properties DiskProperties `json:"properties,omitempty"`

	Tags map[string]string `json:"tags,omitempty"`
}

func (self *SRegion) CreateDisk(storageType string, name string, sizeGb int32, desc string, imageId string) (*SDisk, error) {
	disk := SDisk{
		Name:     name,
		Location: self.Name,
		Sku: DiskSku{
			Name: storageType,
		},
		Properties: DiskProperties{
			CreationData: CreationData{
				CreateOption: "Empty",
			},
			DiskSizeGB: sizeGb,
		},
		Type: "Microsoft.Compute/disks",
	}
	if len(imageId) > 0 {
		image, err := self.GetImage(imageId)
		if err != nil {
			return nil, err
		}
		blobUrl := image.GetBlobUri()
		if len(blobUrl) == 0 {
			return nil, fmt.Errorf("failed to find blobUri for image %s", image.Name)
		}
		disk.Properties.CreationData = CreationData{
			CreateOption: "Import",
			SourceURI:    blobUrl,
		}
		disk.Properties.OsType = image.GetOsType()
	}
	return &disk, self.client.Create(jsonutils.Marshal(disk), &disk)
}

func (self *SRegion) DeleteDisk(diskId string) error {
	return self.deleteDisk(diskId)
}

func (self *SRegion) deleteDisk(diskId string) error {
	if !strings.HasPrefix(diskId, "https://") {
		return self.client.Delete(diskId)
	}
	//TODO
	return cloudprovider.ErrNotImplemented
}

func (self *SRegion) ResizeDisk(diskId string, sizeGb int32) error {
	if !strings.HasPrefix(diskId, "https://") {
		disk, err := self.GetDisk(diskId)
		if err != nil {
			return err
		}
		disk.Properties.DiskSizeGB = sizeGb
		disk.Properties.ProvisioningState = ""
		return self.client.Update(jsonutils.Marshal(disk), nil)
	}
	return cloudprovider.ErrNotSupported
}

func (self *SRegion) GetDisk(diskId string) (*SDisk, error) {
	disk := SDisk{}
	return &disk, self.client.Get(diskId, []string{}, &disk)
}

func (self *SRegion) GetDisks() ([]SDisk, error) {
	result := []SDisk{}
	disks := []SDisk{}
	err := self.client.ListAll("Microsoft.Compute/disks", &disks)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(disks); i++ {
		if disks[i].Location == self.Name {
			result = append(result, disks[i])
		}
	}
	return result, nil
}

func (self *SDisk) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(models.HYPERVISOR_AZURE), "hypervisor")
	return data
}

func (self *SDisk) GetStatus() string {
	if !strings.HasPrefix(self.ID, "https://") {
		status := self.Properties.ProvisioningState
		switch status {
		case "Updating":
			return models.DISK_ALLOCATING
		case "Succeeded":
			return models.DISK_READY
		default:
			log.Errorf("Unknow azure disk %s status: %s", self.ID, status)
			return models.DISK_UNKNOWN
		}
	}
	return models.DISK_READY
}

func (self *SDisk) GetId() string {
	return self.ID
}

func (self *SDisk) Refresh() error {
	if !strings.HasPrefix(self.ID, "https://") {
		disk, err := self.storage.zone.region.GetDisk(self.ID)
		if err != nil {
			return cloudprovider.ErrNotFound
		}
		return jsonutils.Update(self, disk)
	}
	return nil
}

func (self *SDisk) Delete(ctx context.Context) error {
	return self.storage.zone.region.deleteDisk(self.ID)
}

func (self *SDisk) Resize(ctx context.Context, sizeMb int64) error {
	return self.storage.zone.region.ResizeDisk(self.ID, int32(sizeMb/1024))
}

func (self *SDisk) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	return self.ID
}

func (self *SDisk) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SDisk) IsEmulated() bool {
	return false
}

func (self *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return self.storage, nil
}

func (self *SDisk) GetFsFormat() string {
	return ""
}

func (self *SDisk) GetIsNonPersistent() bool {
	return false
}

func (self *SDisk) GetDriver() string {
	return "scsi"
}

func (self *SDisk) GetCacheMode() string {
	return "none"
}

func (self *SDisk) GetMountpoint() string {
	return ""
}

func (self *SDisk) GetDiskFormat() string {
	return "vhd"
}

func (self *SDisk) GetDiskSizeMB() int {
	return int(self.Properties.DiskSizeGB) * 1024
}

func (self *SDisk) GetIsAutoDelete() bool {
	return false
}

func (self *SDisk) GetTemplateId() string {
	if self.Properties.CreationData.ImageReference != nil {
		return self.Properties.CreationData.ImageReference.ID
	}
	return ""
}

func (self *SDisk) GetDiskType() string {
	if len(self.Properties.OsType) > 0 {
		return models.DISK_TYPE_SYS
	}
	return models.DISK_TYPE_DATA
}

func (self *SDisk) CreateISnapshot(ctx context.Context, name, desc string) (cloudprovider.ICloudSnapshot, error) {
	if snapshot, err := self.storage.zone.region.CreateSnapshot(self.ID, name, desc); err != nil {
		log.Errorf("createSnapshot fail %s", err)
		return nil, err
	} else {
		return snapshot, nil
	}
}

func (self *SDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return self.GetSnapshotDetail(snapshotId)
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	isnapshots := make([]cloudprovider.ICloudSnapshot, 0)
	if !strings.HasPrefix(self.ID, "https://") {
		snapshots, err := self.storage.zone.region.GetSnapShots(self.ID)
		if err != nil {
			return nil, err
		}

		for i := 0; i < len(snapshots); i++ {
			isnapshots = append(isnapshots, &snapshots[i])
		}
	}
	return isnapshots, nil
}

func (self *SDisk) GetBillingType() string {
	return models.BILLING_TYPE_POSTPAID
}

func (self *SDisk) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SDisk) GetSnapshotDetail(snapshotId string) (*SSnapshot, error) {
	snapshot, err := self.storage.zone.region.GetSnapshotDetail(snapshotId)
	if err != nil {
		return nil, err
	}
	if snapshot.Properties.CreationData.SourceResourceID != self.ID {
		return nil, cloudprovider.ErrNotFound
	}
	return snapshot, nil
}

func (region *SRegion) GetSnapshotDetail(snapshotId string) (*SSnapshot, error) {
	snapshot := SSnapshot{region: region}
	return &snapshot, region.client.Get(snapshotId, []string{}, &snapshot)
}

func (region *SRegion) GetSnapShots(diskId string) ([]SSnapshot, error) {
	result := []SSnapshot{}
	if !strings.HasPrefix(diskId, "https://") {
		snapshots := []SSnapshot{}
		err := region.client.ListAll("Microsoft.Compute/snapshots", &snapshots)
		if err != nil {
			return nil, err
		}
		for i := 0; i < len(snapshots); i++ {
			if snapshots[i].Location == region.Name {
				if len(diskId) == 0 || diskId == snapshots[i].Properties.CreationData.SourceResourceID {
					snapshots[i].region = region
					result = append(result, snapshots[i])
				}
			}
		}
	}
	return result, nil
}

func (self *SDisk) Reset(ctx context.Context, snapshotId string) error {
	return self.storage.zone.region.resetDisk(self.ID, snapshotId)
}

func (self *SRegion) resetDisk(diskId, snapshotId string) error {
	// TODO
	return cloudprovider.ErrNotSupported
}

func (disk *SDisk) GetAccessPath() string {
	return ""
}

func (self *SDisk) Rebuild(ctx context.Context) error {
	// TODO
	return cloudprovider.ErrNotSupported
}

package azure

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
)

type StorageAccountTypes string

const (
	// StorageAccountTypesPremiumLRS ...
	StorageAccountTypesPremiumLRS StorageAccountTypes = "Premium_LRS"
	// StorageAccountTypesStandardLRS ...
	StorageAccountTypesStandardLRS StorageAccountTypes = "Standard_LRS"
	// StorageAccountTypesStandardSSDLRS ...
	StorageAccountTypesStandardSSDLRS StorageAccountTypes = "StandardSSD_LRS"
)

type DiskSku struct {
	Name StorageAccountTypes
	Tier string
}

type ImageDiskReference struct {
	ID  string
	Lun int32
}

type CreationData struct {
	CreateOption     string
	StorageAccountID string
	ImageReference   ImageDiskReference
	SourceURI        string
	SourceResourceID string
}

type DiskProperties struct {
	TimeCreated       time.Time
	OsType            OperatingSystemTypes
	CreationData      CreationData
	DiskSizeGB        int32
	ProvisioningState string
}

type SDisk struct {
	storage *SStorage

	ManagedBy  string
	Sku        DiskSku
	Zones      []string
	ID         string
	Name       string
	Type       string
	Location   string
	Properties DiskProperties

	Tags map[string]string
}

func (self *SRegion) CreateDisk(storageType string, name string, sizeGb int32, desc string, imageId string) (string, error) {
	return self.createDisk(storageType, name, sizeGb, desc, imageId)
}

func (self *SRegion) createDisk(storageType string, name string, sizeGb int32, desc string, imageId string) (string, error) {
	computeClient := compute.NewDisksClientWithBaseURI(self.client.baseUrl, self.client.subscriptionId)
	computeClient.Authorizer = self.client.authorizer
	sku := compute.DiskSku{Name: compute.StorageAccountTypes(storageType)}
	properties := compute.DiskProperties{DiskSizeGB: &sizeGb, CreationData: &compute.CreationData{CreateOption: compute.Empty}}
	if len(imageId) > 0 {
		globalId, _, _ := pareResourceGroupWithName(imageId, IMAGE_RESOURCE)
		if image, err := self.GetImage(globalId); err != nil {
			return "", err
		} else if blobUri := image.GetBlobUri(); len(blobUri) == 0 {
			return "", fmt.Errorf("failed to find blobUri for image %s", image.Name)
		} else {
			properties.CreationData = &compute.CreationData{
				CreateOption: compute.Import,
				SourceURI:    &blobUri,
			}
			properties.OsType = compute.OperatingSystemTypes(image.GetOsType())
		}
	}
	disk := compute.Disk{Name: &name, Location: &self.Name, DiskProperties: &properties, Sku: &sku}
	diskId, resourceGroup, diskName := pareResourceGroupWithName(name, DISK_RESOURCE)
	//log.Debugf("Create disk: %s", jsonutils.Marshal(disk).PrettyString())
	if result, err := computeClient.CreateOrUpdate(context.Background(), resourceGroup, diskName, disk); err != nil {
		return "", err
	} else if err := result.WaitForCompletion(context.Background(), computeClient.Client); err != nil {
		return "", err
	} else {
		return diskId, nil
	}
}

func (self *SRegion) DeleteDisk(diskId string) error {
	return self.deleteDisk(diskId)
}

func (self *SRegion) deleteDisk(diskId string) error {
	diskClient := compute.NewDisksClientWithBaseURI(self.client.baseUrl, self.client.subscriptionId)
	diskClient.Authorizer = self.client.authorizer
	_, resourceGroup, name := pareResourceGroupWithName(diskId, DISK_RESOURCE)
	if result, err := diskClient.Delete(context.Background(), resourceGroup, name); err != nil {
		return err
	} else if err := result.WaitForCompletion(context.Background(), diskClient.Client); err != nil {
		return err
	}
	return nil
}

func (self *SRegion) ResizeDisk(diskId string, sizeGb int32) error {
	return self.resizeDisk(diskId, sizeGb)
}

func (self *SRegion) resizeDisk(diskId string, sizeGb int32) error {
	diskClient := compute.NewDisksClientWithBaseURI(self.client.baseUrl, self.client.subscriptionId)
	diskClient.Authorizer = self.client.authorizer
	_, resourceGroup, diskName := pareResourceGroupWithName(diskId, DISK_RESOURCE)
	params := compute.DiskUpdate{
		DiskUpdateProperties: &compute.DiskUpdateProperties{
			DiskSizeGB: &sizeGb,
		},
	}
	if result, err := diskClient.Update(context.Background(), resourceGroup, diskName, params); err != nil {
		return err
	} else if err := result.WaitForCompletion(context.Background(), diskClient.Client); err != nil {
		return err
	}
	return nil
}

func (self *SRegion) GetDisk(diskId string) (*SDisk, error) {
	disk := SDisk{}
	computeClient := compute.NewDisksClientWithBaseURI(self.client.baseUrl, self.client.subscriptionId)
	computeClient.Authorizer = self.client.authorizer
	_, resourceGroup, diskName := pareResourceGroupWithName(diskId, DISK_RESOURCE)
	if len(diskId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if _disk, err := computeClient.Get(context.Background(), resourceGroup, diskName); err != nil {
		if _disk.Response.StatusCode == 404 {
			return nil, cloudprovider.ErrNotFound
		}
		return nil, err
	} else if err := jsonutils.Update(&disk, _disk); err != nil {
		return nil, err
	} else {
		return &disk, nil
	}
}

func (self *SRegion) GetDisks() ([]SDisk, error) {
	disks := make([]SDisk, 0)
	computeClient := compute.NewDisksClientWithBaseURI(self.client.baseUrl, self.client.subscriptionId)
	computeClient.Authorizer = self.client.authorizer
	if diskList, err := computeClient.List(context.Background()); err != nil {
		return nil, err
	} else {
		for _, _disk := range diskList.Values() {
			if *_disk.Location == self.Name {
				disk := SDisk{}
				if err := jsonutils.Update(&disk, _disk); err != nil {
					return disks, err
				}
				disks = append(disks, disk)
			}
		}
	}
	return disks, nil
}

func (self *SDisk) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(models.HYPERVISOR_AZURE), "hypervisor")
	return data
}

func (self *SDisk) GetStatus() string {
	status := self.Properties.ProvisioningState
	switch status {
	case "Updating":
		return models.DISK_ALLOCATING
	case "Succeeded":
		return models.DISK_READY
	default:
		log.Errorf("Unknow azure disk status: %s", status)
		return models.DISK_UNKNOWN
	}
}

func (self *SDisk) GetId() string {
	return self.ID
}

func (self *SDisk) Refresh() error {
	if disk, err := self.storage.zone.region.GetDisk(self.ID); err != nil {
		return cloudprovider.ErrNotFound
	} else {
		return jsonutils.Update(self, disk)
	}
}

func (self *SDisk) Delete() error {
	return self.storage.zone.region.deleteDisk(self.ID)
}

func (self *SDisk) Resize(size int64) error {
	return self.storage.zone.region.resizeDisk(self.ID, int32(size))
}

func (self *SDisk) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	return self.ID
}

func (self *SDisk) GetGlobalId() string {
	return self.ID
}

func (self *SDisk) IsEmulated() bool {
	return false
}

func (self *SDisk) GetIStorge() cloudprovider.ICloudStorage {
	return self.storage
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
	return self.Properties.CreationData.ImageReference.ID
}

func (self *SDisk) GetDiskType() string {
	if len(self.Properties.OsType) > 0 {
		return models.DISK_TYPE_SYS
	}
	return models.DISK_TYPE_DATA
}

func (self *SDisk) CreateISnapshot(name, desc string) (cloudprovider.ICloudSnapshot, error) {
	if snapshot, err := self.storage.zone.region.CreateSnapshot(self.ID, name, desc); err != nil {
		log.Errorf("createSnapshot fail %s", err)
		return nil, err
	} else {
		snapshot.disk = self
		return snapshot, nil
	}
}

func (self *SDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return self.GetSnapshotDetail(snapshotId)
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	if snapshots, err := self.storage.zone.region.GetSnapShots(self.ID); err != nil {
		return nil, err
	} else {
		isnapshots := make([]cloudprovider.ICloudSnapshot, len(snapshots))
		for i := 0; i < len(snapshots); i++ {
			snapshots[i].disk = self
			isnapshots[i] = &snapshots[i]
		}
		return isnapshots, nil
	}
}

func (self *SDisk) GetBillingType() string {
	return models.BILLING_TYPE_POSTPAID
}

func (self *SDisk) GetExpiredAt() time.Time {
	return time.Now()
}

func (self *SDisk) GetSnapshotDetail(snapshotId string) (*SSnapshot, error) {
	if snapshot, err := self.storage.zone.region.GetSnapshotDetail(snapshotId); err != nil {
		return nil, err
	} else if snapshot.Properties.CreationData.SourceResourceID != self.ID {
		return nil, cloudprovider.ErrNotFound
	} else {
		snapshot.disk = self
		return snapshot, nil
	}
}

func (region *SRegion) GetSnapshotDetail(snapshotId string) (*SSnapshot, error) {
	snapshot := SSnapshot{}
	_, resourceGroup, snapshotName := pareResourceGroupWithName(snapshotId, SNAPSHOT_RESOURCE)
	snapClient := compute.NewSnapshotsClientWithBaseURI(region.client.baseUrl, region.SubscriptionID)
	snapClient.Authorizer = region.client.authorizer
	if result, err := snapClient.Get(context.Background(), resourceGroup, snapshotName); err != nil {
		if result.Response.StatusCode == 404 {
			return nil, cloudprovider.ErrNotFound
		}
		return nil, err
	} else if err := jsonutils.Update(&snapshot, result); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (region *SRegion) GetSnapShots(diskId string) ([]SSnapshot, error) {
	snapshots := []SSnapshot{}
	globalId, _, _ := pareResourceGroupWithName(diskId, DISK_RESOURCE)
	snapClient := compute.NewSnapshotsClientWithBaseURI(region.client.baseUrl, region.SubscriptionID)
	snapClient.Authorizer = region.client.authorizer
	if result, err := snapClient.List(context.Background()); err != nil {
		return nil, err
	} else if len(diskId) > 0 {
		data := result.Values()
		for i := 0; i < len(data); i++ {
			snap := SSnapshot{}
			_globalId, _, _ := pareResourceGroupWithName(*data[i].CreationData.SourceResourceID, DISK_RESOURCE)
			if globalId == _globalId {
				if err := jsonutils.Update(&snap, data[i]); err != nil {
					return nil, err
				}
				snapshots = append(snapshots, snap)
			}
		}
	} else if err := jsonutils.Update(&snapshots, result.Values()); err != nil {
		return snapshots, nil
	}
	return snapshots, nil
}

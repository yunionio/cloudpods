package azure

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
)

type DiskSku struct {
	Name string
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

	ResourceGroup string

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

func (self *SRegion) GetDisk(resourceGroup string, diskName string) (*SDisk, error) {
	disk := SDisk{}
	computeClient := compute.NewDisksClientWithBaseURI(self.client.baseUrl, self.client.subscriptionId)
	computeClient.Authorizer = self.client.authorizer
	if _disk, err := computeClient.Get(context.Background(), resourceGroup, diskName); err != nil {
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
			disk := SDisk{}
			if err := jsonutils.Update(&disk, _disk); err != nil {
				return disks, err
			}
			disk.ResourceGroup, _, _ = pareResourceGroupWithName(disk.ID)
			disks = append(disks, disk)
		}
	}
	return disks, nil
}

func (self *SDisk) GetStatus() string {
	// In_use Available Attaching Detaching Creating ReIniting All
	switch self.Properties.ProvisioningState {
	case "Updating":
		return models.DISK_ALLOCATING
	default:
		return models.DISK_READY
	}
}

func (self *SDisk) GetId() string {
	return self.ID
}

func (self *SRegion) getDisk(resourceGroup string, diskName string) (*SDisk, error) {
	return self.GetDisk(resourceGroup, diskName)
}

func (self *SDisk) Refresh() error {
	if disk, err := self.storage.zone.region.GetDisk(self.ResourceGroup, self.Name); err != nil {
		return err
	} else {
		return jsonutils.Update(self, disk)
	}
}

func (self *SDisk) Delete() error {
	return nil
	//return self.storage.zone.region.deleteDisk(self.DiskId)
}

func (self *SDisk) Resize(size int64) error {
	//return self.storage.zone.region.resizeDisk(self.DiskId, size)
	return nil
}

func (self *SDisk) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	return self.ID
}

func (self *SDisk) GetGlobalId() string {
	resourceGroup, _, _ := pareResourceGroupWithName(self.ID)
	return fmt.Sprintf("resourceGroups/%s/providers/%s/%s", resourceGroup, self.storage.zone.region.SubscriptionID, self.Name)
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

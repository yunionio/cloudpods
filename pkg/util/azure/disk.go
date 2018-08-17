package azure

import (
	"context"

	"yunion.io/x/jsonutils"
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

type SDisk struct {
	storage *SStorage

	DiskName           string
	DiskId             string
	Size               int32
	DeleteWithInstance bool
	ImageId            string
	OsType             string
	Status             string
	resourceGroup      string

	ManagedBy string
	Sku       DiskSku
	Zones     []string
	ID        string
	Name      string
	Type      string
	Location  string

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
			disk.resourceGroup, _, _ = pareResourceGroupWithName(disk.ID)
			disks = append(disks, disk)
		}
	}
	return disks, nil
}

func (self *SDisk) GetStatus() string {
	// In_use Available Attaching Detaching Creating ReIniting All
	switch self.Status {
	case "Creating", "ReIniting":
		return models.DISK_ALLOCATING
	default:
		return models.DISK_READY
	}
}

func (self *SDisk) GetId() string {
	return self.DiskId
}

func (self *SRegion) getDisk(resourceGroup string, diskName string) (*SDisk, error) {
	return self.GetDisk(resourceGroup, diskName)
}

func (self *SDisk) Refresh() error {
	if disk, err := self.storage.zone.region.GetDisk(self.resourceGroup, self.Name); err != nil {
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
	if len(self.DiskName) > 0 {
		return self.DiskName
	}
	return self.DiskId
}

func (self *SDisk) GetGlobalId() string {
	return self.DiskId
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
	return int(self.Size) * 1024
}

func (self *SDisk) GetIsAutoDelete() bool {
	return self.DeleteWithInstance
}

func (self *SDisk) GetTemplateId() string {
	return self.ImageId
}

func (self *SDisk) GetDiskType() string {
	if len(self.OsType) > 0 {
		return models.DISK_TYPE_SYS
	}
	return models.DISK_TYPE_DATA
}

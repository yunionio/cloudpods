package hostdrivers

import (
	"fmt"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SUCloudHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SUCloudHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SUCloudHostDriver) GetHostType() string {
	return models.HOST_TYPE_UCLOUD
}

func (self *SUCloudHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	if sizeGb > 2000 {
		return fmt.Errorf("The %s disk size must be in the range of 1 ~ 2000GB", storage.StorageType)
	}

	return nil
}

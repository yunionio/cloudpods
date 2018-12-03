package hostdrivers

import (
	"fmt"

	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/compute/models"
)

type SAliyunHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SAliyunHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SAliyunHostDriver) GetHostType() string {
	return models.HOST_TYPE_ALIYUN
}

func (self *SAliyunHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	if utils.IsInStringArray(storage.StorageType, []string{models.STORAGE_CLOUD_EFFICIENCY, models.STORAGE_CLOUD_SSD, models.STORAGE_CLOUD_ESSD}) {
		if sizeGb < 20 || sizeGb > 32768 {
			return fmt.Errorf("The %s disk size must be in the range of 20G ~ 32768GB", storage.StorageType)
		}
	} else if storage.StorageType == models.STORAGE_PUBLIC_CLOUD {
		if sizeGb < 5 || sizeGb > 2000 {
			return fmt.Errorf("The %s disk size must be in the range of 5G ~ 2000GB", storage.StorageType)
		}
	} else {
		return fmt.Errorf("Not support create %s disk", storage.StorageType)
	}
	return nil
}

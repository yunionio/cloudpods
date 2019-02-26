package hostdrivers

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type SAwsHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SAwsHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SAwsHostDriver) GetHostType() string {
	return models.HOST_TYPE_AWS
}

func (self *SAwsHostDriver) ValidateAttachStorage(host *models.SHost, storage *models.SStorage, data *jsonutils.JSONDict) error {
	return httperrors.NewUnsupportOperationError("Not support attach storage for %s host", self.GetHostType())
}

func (self *SAwsHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	if storage.StorageType == models.STORAGE_GP2_SSD {
		if sizeGb < 1 || sizeGb > 16384 {
			return fmt.Errorf("The %s disk size must be in the range of 1G ~ 16384GB", storage.StorageType)
		}
	} else if storage.StorageType == models.STORAGE_IO1_SSD {
		if sizeGb < 4 || sizeGb > 16384 {
			return fmt.Errorf("The %s disk size must be in the range of 4G ~ 16384GB", storage.StorageType)
		}
	} else if utils.IsInStringArray(storage.StorageType, []string{models.STORAGE_ST1_HDD, models.STORAGE_SC1_HDD}) {
		if sizeGb < 500 || sizeGb > 16384 {
			return fmt.Errorf("The %s disk size must be in the range of 500G ~ 16384GB", storage.StorageType)
		}
	} else if storage.StorageType == models.STORAGE_STANDARD_HDD {
		if sizeGb < 1 || sizeGb > 1024 {
			return fmt.Errorf("The %s disk size must be in the range of 1G ~ 1024GB", storage.StorageType)
		}
	} else {
		return fmt.Errorf("Not support create %s disk", storage.StorageType)
	}
	return nil
}

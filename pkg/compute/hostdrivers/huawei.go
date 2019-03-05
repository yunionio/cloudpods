package hostdrivers

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type SHuaweiHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SHuaweiHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SHuaweiHostDriver) GetHostType() string {
	return models.HOST_TYPE_HUAWEI
}

func (self *SHuaweiHostDriver) ValidateAttachStorage(host *models.SHost, storage *models.SStorage, data *jsonutils.JSONDict) error {
	return httperrors.NewUnsupportOperationError("Not support attach storage for %s host", self.GetHostType())
}

// 系统盘必须至少40G
func (self *SHuaweiHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	switch storage.StorageType {
	case models.STORAGE_HUAWEI_SSD, models.STORAGE_HUAWEI_SATA, models.STORAGE_HUAWEI_SAS:
		if sizeGb < 10 || sizeGb > 32768 {
			return fmt.Errorf("The %s disk size must be in the range of 10G ~ 32768GB", storage.StorageType)
		}
	default:
		return fmt.Errorf("Not support create %s disk", storage.StorageType)
	}

	return nil
}

package hostdrivers

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type SQcloudHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SQcloudHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SQcloudHostDriver) GetHostType() string {
	return models.HOST_TYPE_QCLOUD
}

func (self *SQcloudHostDriver) ValidateAttachStorage(host *models.SHost, storage *models.SStorage, data *jsonutils.JSONDict) error {
	return httperrors.NewUnsupportOperationError("Not support attach storage for %s host", self.GetHostType())
}

func (self *SQcloudHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	if sizeGb%10 != 0 {
		return fmt.Errorf("The disk size must be a multiple of 10Gb")
	}
	if storage.StorageType == models.STORAGE_CLOUD_BASIC {
		if sizeGb < 10 || sizeGb > 16000 {
			return fmt.Errorf("The %s disk size must be in the range of 10 ~ 16000GB", storage.StorageType)
		}
	} else if storage.StorageType == models.STORAGE_CLOUD_PREMIUM {
		if sizeGb < 50 || sizeGb > 16000 {
			return fmt.Errorf("The %s disk size must be in the range of 50 ~ 16000GB", storage.StorageType)
		}
	} else if storage.StorageType == models.STORAGE_CLOUD_SSD {
		if sizeGb < 100 || sizeGb > 16000 {
			return fmt.Errorf("The %s disk size must be in the range of 100 ~ 16000GB", storage.StorageType)
		}
	} else {
		return fmt.Errorf("Not support create %s disk", storage.StorageType)
	}
	return nil
}

func (self *SQcloudHostDriver) RequestDeleteSnapshotWithStorage(ctx context.Context, host *models.SHost, snapshot *models.SSnapshot, task taskman.ITask) error {
	return httperrors.NewNotImplementedError("not implement")
}

func (driver *SQcloudHostDriver) GetStoragecacheQuota(host *models.SHost) int {
	return 10
}

package hostdrivers

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SAzureHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SAzureHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SAzureHostDriver) GetHostType() string {
	return models.HOST_TYPE_AZURE
}

func (self *SAzureHostDriver) ValidateAttachStorage(host *models.SHost, storage *models.SStorage, data *jsonutils.JSONDict) error {
	return httperrors.NewUnsupportOperationError("Not support attach storage for %s host", self.GetHostType())
}

func (self *SAzureHostDriver) ValidateUpdateDisk(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if data.Contains("name") {
		return nil, httperrors.NewInputParameterError("cannot support change azure disk name")
	}
	return data, nil
}

func (self *SAzureHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	if utils.IsInStringArray(storage.StorageType, []string{models.STORAGE_STANDARD_LRS, models.STORAGE_STANDARDSSD_LRS, models.STORAGE_PREMIUM_LRS}) {
		if sizeGb < 1 || sizeGb > 4095 {
			return fmt.Errorf("The %s disk size must be in the range of 1G ~ 4095GB", storage.StorageType)
		}
	} else {
		return fmt.Errorf("Not support create %s disk", storage.StorageType)
	}
	return nil
}

func (self *SAzureHostDriver) RequestDeleteSnapshotWithStorage(ctx context.Context, host *models.SHost, snapshot *models.SSnapshot, task taskman.ITask) error {
	return httperrors.NewNotImplementedError("not implement")
}

package hostdrivers

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type SBaremetalHostDriver struct {
	SBaseHostDriver
}

func init() {
	driver := SBaremetalHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SBaremetalHostDriver) GetHostType() string {
	return models.HOST_TYPE_BAREMETAL
}

func (self *SBaremetalHostDriver) ValidateAttachStorage(host *models.SHost, storage *models.SStorage, data *jsonutils.JSONDict) error {
	return httperrors.NewUnsupportOperationError("Not support attach storage for %s host", self.GetHostType())
}

func (self *SBaremetalHostDriver) CheckAndSetCacheImage(ctx context.Context, host *models.SHost, storageCache *models.SStoragecache, task taskman.ITask) error {
	return fmt.Errorf("not supported")
}

func (self *SBaremetalHostDriver) RequestAllocateDiskOnStorage(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask, content *jsonutils.JSONDict) error {
	return fmt.Errorf("not supported")
}

func (self *SBaremetalHostDriver) RequestDeallocateDiskOnHost(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask) error {
	return fmt.Errorf("not supported")
}

func (self *SBaremetalHostDriver) RequestPrepareSaveDiskOnHost(ctx context.Context, host *models.SHost, disk *models.SDisk, imageId string, task taskman.ITask) error {
	return fmt.Errorf("not supported")
}

func (self *SBaremetalHostDriver) RequestRebuildDiskOnStorage(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask, content *jsonutils.JSONDict) error {
	return fmt.Errorf("not supported")
}

func (self *SBaremetalHostDriver) RequestResizeDiskOnHost(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, guest *models.SGuest, sizeMb int64, task taskman.ITask) error {
	return fmt.Errorf("not supported")
}

func (self *SBaremetalHostDriver) RequestSaveUploadImageOnHost(ctx context.Context, host *models.SHost, disk *models.SDisk, imageId string, task taskman.ITask, data jsonutils.JSONObject) error {
	return fmt.Errorf("not supported")
}

func (self *SBaremetalHostDriver) RequestUncacheImage(ctx context.Context, host *models.SHost, storageCache *models.SStoragecache, task taskman.ITask) error {
	return fmt.Errorf("not supported")
}

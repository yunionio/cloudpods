package hostdrivers

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type SOpenStackHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SOpenStackHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SOpenStackHostDriver) GetHostType() string {
	return models.HOST_TYPE_OPENSTACK
}

func (self *SOpenStackHostDriver) ValidateAttachStorage(host *models.SHost, storage *models.SStorage, data *jsonutils.JSONDict) error {
	return httperrors.NewUnsupportOperationError("Not support attach storage for %s host", self.GetHostType())
}

func (self *SOpenStackHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	return nil
}

func (driver *SOpenStackHostDriver) GetStoragecacheQuota(host *models.SHost) int {
	return 100
}

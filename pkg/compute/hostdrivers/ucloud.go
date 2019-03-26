package hostdrivers

import "yunion.io/x/onecloud/pkg/compute/models"

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

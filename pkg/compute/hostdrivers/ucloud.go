package hostdrivers

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
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
	return api.HOST_TYPE_UCLOUD
}

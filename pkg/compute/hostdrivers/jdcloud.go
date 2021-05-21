package hostdrivers

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SJDcloudHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SJDcloudHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SJDcloudHostDriver) GetHostType() string {
	return api.HOST_TYPE_JDCLOUD
}

func (self *SJDcloudHostDriver) GetHypervisor() string {
	return api.HYPERVISOR_JDCLOUD
}

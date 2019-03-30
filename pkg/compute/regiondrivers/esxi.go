package regiondrivers

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SEsxiRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SEsxiRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SEsxiRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_VMWARE
}

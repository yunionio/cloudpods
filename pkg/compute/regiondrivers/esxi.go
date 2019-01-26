package regiondrivers

import (
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
	return models.CLOUD_PROVIDER_VMWARE
}

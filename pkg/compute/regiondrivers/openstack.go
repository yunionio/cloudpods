package regiondrivers

import (
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SOpenStackRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SOpenStackRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SOpenStackRegionDriver) GetProvider() string {
	return models.CLOUD_PROVIDER_OPENSTACK
}

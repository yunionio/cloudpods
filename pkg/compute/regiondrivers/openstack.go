package regiondrivers

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
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
	return api.CLOUD_PROVIDER_OPENSTACK
}

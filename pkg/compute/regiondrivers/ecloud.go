package regiondrivers

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SEcloudRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SEcloudRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SEcloudRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_ECLOUD
}

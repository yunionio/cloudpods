package regiondrivers

import (
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SHuaWeiRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SHuaWeiRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SHuaWeiRegionDriver) GetProvider() string {
	return models.CLOUD_PROVIDER_HUAWEI
}

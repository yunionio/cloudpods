package regiondrivers

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
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
	return api.CLOUD_PROVIDER_HUAWEI
}

package regiondrivers

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SHuaweiCloudStackRegionDriver struct {
	SHuaWeiRegionDriver
}

func init() {
	driver := SHuaweiCloudStackRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SHuaweiCloudStackRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_HUAWEI_CLOUD_STACK
}

func (self *SHuaweiCloudStackRegionDriver) IsSupportedElasticcache() bool {
	return false
}

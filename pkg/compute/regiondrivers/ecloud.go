package regiondrivers

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/pinyinutils"
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

func (self *SEcloudRegionDriver) IsAllowSecurityGroupNameRepeat() bool {
	return false
}

func (self *SEcloudRegionDriver) GenerateSecurityGroupName(name string) string {
	return pinyinutils.Text2Pinyin(name)
}

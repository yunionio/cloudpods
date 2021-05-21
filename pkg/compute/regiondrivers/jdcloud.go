package regiondrivers

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SJDcloudRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SJDcloudRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SJDcloudRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_JDCLOUD
}

// https://docs.jdcloud.com/cn/ssl-certificate/api/overview
func (self *SJDcloudRegionDriver) IsCertificateBelongToRegion() bool {
	return false
}

// https://docs.jdcloud.com/cn/virtual-private-cloud/api/modifynetworksecuritygrouprules
func (self *SJDcloudRegionDriver) IsOnlySupportAllowRules() bool {
	return true
}

func (self *SJDcloudRegionDriver) IsSecurityGroupBelongVpc() bool {
	return true
}

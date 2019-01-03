package regiondrivers

import (
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SQcloudRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SQcloudRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SQcloudRegionDriver) GetProvider() string {
	return models.CLOUD_PROVIDER_QCLOUD
}

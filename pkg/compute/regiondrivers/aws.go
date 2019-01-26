package regiondrivers

import (
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SAwsRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SAwsRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SAwsRegionDriver) GetProvider() string {
	return models.CLOUD_PROVIDER_AWS
}

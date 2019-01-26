package regiondrivers

import (
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SAzureRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SAzureRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SAzureRegionDriver) GetProvider() string {
	return models.CLOUD_PROVIDER_AZURE
}

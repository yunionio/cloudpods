package hostdrivers

import (
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SAwsHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SAwsHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SAwsHostDriver) GetHostType() string {
	return models.HOST_TYPE_AWS
}

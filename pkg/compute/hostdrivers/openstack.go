package hostdrivers

import (
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SOpenStackHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SOpenStackHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SOpenStackHostDriver) GetHostType() string {
	return models.HOST_TYPE_OPENSTACK
}

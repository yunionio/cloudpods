package guestdrivers

import (
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SOpenStackGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SOpenStackGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SOpenStackGuestDriver) GetHypervisor() string {
	return models.HYPERVISOR_OPENSTACK
}

func (self *SOpenStackGuestDriver) IsSupportEip() bool {
	return false
}

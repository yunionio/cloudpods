package guestdrivers

import (
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SESXiGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SESXiGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SESXiGuestDriver) GetHypervisor() string {
	return models.HYPERVISOR_ESXI
}

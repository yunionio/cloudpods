package guestdrivers

import "yunion.io/x/onecloud/pkg/compute/models"

type SUCloudGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func (self *SUCloudGuestDriver) GetHypervisor() string {
	return models.HYPERVISOR_UCLOUD
}

func (self *SUCloudGuestDriver) GetDefaultSysDiskBackend() string {
	return models.STORAGE_UCLOUD_CLOUD_SSD
}

func (self *SUCloudGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 10
}

func init() {
	driver := SUCloudGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

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

func (self *SUCloudGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{models.VM_READY}, nil
}

func (self *SUCloudGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 10
}

func (self *SUCloudGuestDriver) GetGuestInitialStateAfterCreate() string {
	return models.VM_RUNNING
}

func (self *SUCloudGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SUCloudGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func init() {
	driver := SUCloudGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

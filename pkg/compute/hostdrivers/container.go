package hostdrivers

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
)

func init() {
	driver := &SContainerHostDriver{
		SKVMHostDriver: &SKVMHostDriver{},
	}
	models.RegisterHostDriver(driver)
}

type SContainerHostDriver struct {
	*SKVMHostDriver
}

func (d *SContainerHostDriver) GetHostType() string {
	return api.HOST_TYPE_CONTAINER
}

func (d *SContainerHostDriver) GetHypervisor() string {
	return api.HYPERVISOR_POD
}

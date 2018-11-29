package hostdrivers

import (
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SAliyunHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SAliyunHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SAliyunHostDriver) GetHostType() string {
	return models.HOST_TYPE_ALIYUN
}

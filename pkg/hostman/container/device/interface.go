package device

import (
	"fmt"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/onecloud/pkg/apis"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
)

var (
	drivers = make(map[apis.ContainerDeviceType]IDeviceDriver)
)

func RegisterDriver(drv IDeviceDriver) {
	drivers[drv.GetType()] = drv
}

func GetDriver(typ apis.ContainerDeviceType) IDeviceDriver {
	drv, ok := drivers[typ]
	if !ok {
		panic(fmt.Sprintf("not found driver by type %s", typ))
	}
	return drv
}

type IDeviceDriver interface {
	GetType() apis.ContainerDeviceType
	GetRuntimeDevices(input *hostapi.ContainerCreateInput, dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, error)
}

package volume_mount

import (
	"fmt"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
)

var (
	drivers = make(map[apis.ContainerVolumeMountType]IVolumeMount)
)

func RegisterDriver(drv IVolumeMount) {
	drivers[drv.GetType()] = drv
}

func GetDriver(typ apis.ContainerVolumeMountType) IVolumeMount {
	drv, ok := drivers[typ]
	if !ok {
		panic(fmt.Sprintf("not found driver by type %s", typ))
	}
	return drv
}

type IPodInfo interface {
	GetVolumesDir() string
	GetDisks() []*desc.SGuestDisk
	GetDiskMountPoint(disk storageman.IDisk) string
}

type IVolumeMount interface {
	GetType() apis.ContainerVolumeMountType
	GetRuntimeMountHostPath(pod IPodInfo, vm *apis.ContainerVolumeMount) (string, error)
	Mount(pod IPodInfo, vm *apis.ContainerVolumeMount) error
	Unmount(pod IPodInfo, vm *apis.ContainerVolumeMount) error
}

func GetRuntimeVolumeMountPropagation(input apis.ContainerMountPropagation) runtimeapi.MountPropagation {
	switch input {
	case apis.MOUNTPROPAGATION_PROPAGATION_PRIVATE:
		return runtimeapi.MountPropagation_PROPAGATION_PRIVATE
	case apis.MOUNTPROPAGATION_PROPAGATION_HOST_TO_CONTAINER:
		return runtimeapi.MountPropagation_PROPAGATION_HOST_TO_CONTAINER
	case apis.MOUNTPROPAGATION_PROPAGATION_BIDIRECTIONAL:
		return runtimeapi.MountPropagation_PROPAGATION_BIDIRECTIONAL
	}
	// private defaultly
	return runtimeapi.MountPropagation_PROPAGATION_PRIVATE
}

package models

import (
	"context"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	containerVolumeDrivers = make(map[apis.ContainerVolumeMountType]IContainerVolumeMountDriver)
	containerDeviceDrivers = make(map[apis.ContainerDeviceType]IContainerDeviceDriver)
)

func RegisterContainerVolumeMountDriver(drv IContainerVolumeMountDriver) {
	containerVolumeDrivers[drv.GetType()] = drv
}

func GetContainerVolumeMountDriver(typ apis.ContainerVolumeMountType) IContainerVolumeMountDriver {
	drv, err := GetContainerVolumeMountDriverWithError(typ)
	if err != nil {
		panic(err.Error())
	}
	return drv
}

func GetContainerVolumeMountDriverWithError(typ apis.ContainerVolumeMountType) (IContainerVolumeMountDriver, error) {
	drv, ok := containerVolumeDrivers[typ]
	if !ok {
		return nil, httperrors.NewNotFoundError("not found driver by type %q", typ)
	}
	return drv, nil
}

type IContainerVolumeMountDriver interface {
	GetType() apis.ContainerVolumeMountType
	ValidatePodCreateData(ctx context.Context, userCred mcclient.TokenCredential, vm *apis.ContainerVolumeMount, input *api.ServerCreateInput) error
	ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, pod *SGuest, vm *apis.ContainerVolumeMount) (*apis.ContainerVolumeMount, error)
}

func RegisterContainerDeviceDriver(drv IContainerDeviceDriver) {
	containerDeviceDrivers[drv.GetType()] = drv
}

func GetContainerDeviceDriver(typ apis.ContainerDeviceType) IContainerDeviceDriver {
	drv, err := GetContainerDeviceDriverWithError(typ)
	if err != nil {
		panic(err.Error())
	}
	return drv
}

func GetContainerDeviceDriverWithError(typ apis.ContainerDeviceType) (IContainerDeviceDriver, error) {
	drv, ok := containerDeviceDrivers[typ]
	if !ok {
		return nil, httperrors.NewNotFoundError("not found driver by type %q", typ)
	}
	return drv, nil
}

type IContainerDeviceDriver interface {
	GetType() apis.ContainerDeviceType
	ValidatePodCreateData(ctx context.Context, userCred mcclient.TokenCredential, dev *api.ContainerDevice, input *api.ServerCreateInput) error
	ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, pod *SGuest, dev *api.ContainerDevice) (*api.ContainerDevice, error)
	ToHostDevice(dev *api.ContainerDevice) (*hostapi.ContainerDevice, error)
}

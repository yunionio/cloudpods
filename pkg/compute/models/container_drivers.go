// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"context"
	"sync"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	containerVolumeDrivers    = newContainerDrivers()
	containerDeviceDrivers    = newContainerDrivers()
	containerLifecycleDrivers = newContainerDrivers()
)

type containerDrivers struct {
	drivers *sync.Map
}

func newContainerDrivers() *containerDrivers {
	return &containerDrivers{
		drivers: new(sync.Map),
	}
}

func (cd *containerDrivers) GetWithError(typ string) (interface{}, error) {
	drv, ok := cd.drivers.Load(typ)
	if !ok {
		return drv, httperrors.NewNotFoundError("not found driver by type %q", typ)
	}
	return drv, nil
}

func (cd *containerDrivers) Get(typ string) interface{} {
	drv, err := cd.GetWithError(typ)
	if err != nil {
		panic(err.Error())
	}
	return drv
}

func (cd *containerDrivers) Register(typ string, drv interface{}) {
	cd.drivers.Store(typ, drv)
}

func registerContainerDriver[K ~string, D any](drvs *containerDrivers, typ K, drv D) {
	drvs.Register(string(typ), drv)
}

func getContainerDriver[K ~string, D any](drvs *containerDrivers, typ K) D {
	return drvs.Get(string(typ)).(D)
}

func getContainerDriverWithError[K ~string, D any](drvs *containerDrivers, typ K) (D, error) {
	drv, err := drvs.GetWithError(string(typ))
	if err != nil {
		return drv.(D), err
	}
	return drv.(D), nil
}

func RegisterContainerVolumeMountDriver(drv IContainerVolumeMountDriver) {
	registerContainerDriver(containerVolumeDrivers, drv.GetType(), drv)
}

func GetContainerVolumeMountDriver(typ apis.ContainerVolumeMountType) IContainerVolumeMountDriver {
	return getContainerDriver[apis.ContainerVolumeMountType, IContainerVolumeMountDriver](containerVolumeDrivers, typ)
}

func GetContainerVolumeMountDriverWithError(typ apis.ContainerVolumeMountType) (IContainerVolumeMountDriver, error) {
	return getContainerDriverWithError[apis.ContainerVolumeMountType, IContainerVolumeMountDriver](containerVolumeDrivers, typ)
}

type IContainerVolumeMountDriver interface {
	GetType() apis.ContainerVolumeMountType
	ValidatePodCreateData(ctx context.Context, userCred mcclient.TokenCredential, vm *apis.ContainerVolumeMount, input *api.ServerCreateInput) error
	ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, pod *SGuest, vm *apis.ContainerVolumeMount) (*apis.ContainerVolumeMount, error)
}

type IContainerVolumeMountDiskDriver interface {
	IContainerVolumeMountDriver

	ValidatePostOverlay(ctx context.Context, userCred mcclient.TokenCredential, vm *apis.ContainerVolumeMount) error
	ValidatePostSingleOverlay(ctx context.Context, userCred mcclient.TokenCredential, pov *apis.ContainerVolumeMountDiskPostOverlay) error
	ValidatePostOverlayTargetDirs(ovs []*apis.ContainerVolumeMountDiskPostOverlay) error
}

func RegisterContainerDeviceDriver(drv IContainerDeviceDriver) {
	registerContainerDriver(containerDeviceDrivers, drv.GetType(), drv)
}

func GetContainerDeviceDriver(typ apis.ContainerDeviceType) IContainerDeviceDriver {
	return getContainerDriver[apis.ContainerDeviceType, IContainerDeviceDriver](containerDeviceDrivers, typ)
}

func GetContainerDeviceDriverWithError(typ apis.ContainerDeviceType) (IContainerDeviceDriver, error) {
	return getContainerDriverWithError[apis.ContainerDeviceType, IContainerDeviceDriver](containerDeviceDrivers, typ)
}

type IContainerDeviceDriver interface {
	GetType() apis.ContainerDeviceType
	ValidatePodCreateData(ctx context.Context, userCred mcclient.TokenCredential, dev *api.ContainerDevice, input *api.ServerCreateInput) error
	ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, pod *SGuest, dev *api.ContainerDevice) (*api.ContainerDevice, error)
	ToHostDevice(dev *api.ContainerDevice) (*hostapi.ContainerDevice, error)
}

type IContainerLifecyleDriver interface {
	GetType() apis.ContainerLifecyleHandlerType
	ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *apis.ContainerLifecyleHandler) error
}

func RegisterContainerLifecyleDriver(drv IContainerLifecyleDriver) {
	registerContainerDriver(containerLifecycleDrivers, drv.GetType(), drv)
}

func GetContainerLifecyleDriver(typ apis.ContainerLifecyleHandlerType) IContainerLifecyleDriver {
	return getContainerDriver[apis.ContainerLifecyleHandlerType, IContainerLifecyleDriver](containerLifecycleDrivers, typ)
}

func GetContainerLifecyleDriverWithError(typ apis.ContainerLifecyleHandlerType) (IContainerLifecyleDriver, error) {
	return getContainerDriverWithError[apis.ContainerLifecyleHandlerType, IContainerLifecyleDriver](containerLifecycleDrivers, typ)
}

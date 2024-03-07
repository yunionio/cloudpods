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

package isolated_device

import (
	"fmt"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
)

var (
	containerDeviceManagers = make(map[ContainerDeviceType]IContainerDeviceManager)
)

type ContainerDeviceType string

const (
	ContainerDeviceTypeCphAMDGPU     ContainerDeviceType = api.CONTAINER_DEV_CPH_AMD_GPU
	ContainerDeviceTypeCphASOPBinder ContainerDeviceType = api.CONTAINER_DEV_CPH_AOSP_BINDER
	ContainerNetintCAASIC            ContainerDeviceType = api.CONTAINER_DEV_NETINT_CA_ASIC
	ContainerNetintCAQuadra          ContainerDeviceType = api.CONTAINER_DEV_NETINT_CA_QUADRA
	ContainerDeviceTypeNVIDIAGPU     ContainerDeviceType = api.CONTAINER_DEV_NVIDIA_GPU
)

func GetContainerDeviceManager(t ContainerDeviceType) (IContainerDeviceManager, error) {
	man, ok := containerDeviceManagers[t]
	if !ok {
		return nil, errors.Wrapf(errors.ErrNotFound, "not found container device manager by %q", t)
	}
	return man, nil
}

func RegisterContainerDeviceManager(man IContainerDeviceManager) {
	if _, ok := containerDeviceManagers[man.GetType()]; ok {
		panic(fmt.Sprintf("container device manager %s is already registered", man.GetType()))
	}
	containerDeviceManagers[man.GetType()] = man
}

type ContainerDevice struct {
	Path          string              `json:"path"`
	Type          ContainerDeviceType `json:"type"`
	VirtualNumber int                 `json:"virtual_number"`
}

type ContainerDeviceConfiguration struct {
	Devices []*ContainerDevice `json:"devices"`
}

type IContainerDeviceManager interface {
	GetType() ContainerDeviceType
	NewDevices(dev *ContainerDevice) ([]IDevice, error)
	NewContainerDevices(input *hostapi.ContainerCreateInput, dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, error)
	ProbeDevices() ([]IDevice, error)
	GetContainerEnvs(devs []*hostapi.ContainerDevice) []*runtimeapi.KeyValue
}

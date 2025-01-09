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

package volume_mount

import (
	"fmt"

	pdisk "github.com/shirou/gopsutil/v3/disk"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/onecloud/pkg/apis"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
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
	GetName() string
	GetVolumesDir() string
	GetVolumesOverlayDir() string
	GetDisks() []*desc.SGuestDisk
	GetDiskMountPoint(disk storageman.IDisk) string
}

type IVolumeMount interface {
	GetType() apis.ContainerVolumeMountType
	GetRuntimeMountHostPath(pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) (string, error)
	Mount(pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error
	Unmount(pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error
}

type ContainerVolumeMountUsage struct {
	Id         string
	HostPath   string
	MountPath  string
	VolumeType string
	Usage      *pdisk.UsageStat
	Tags       map[string]string
}

type IUsageVolumeMount interface {
	IVolumeMount

	InjectUsageTags(usage *ContainerVolumeMountUsage, vol *hostapi.ContainerVolumeMount)
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

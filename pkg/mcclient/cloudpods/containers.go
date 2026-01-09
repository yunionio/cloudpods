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

package cloudpods

import (
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type SContainer struct {
	multicloud.SResourceBase
	CloudpodsTags
	region *SRegion

	api.SContainer
}

func (region *SRegion) GetContainers(guestId string) ([]SContainer, error) {
	ret := []SContainer{}
	params := map[string]interface{}{
		"guest_id": guestId,
	}
	err := region.list(&modules.Containers, params, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (container *SContainer) GetId() string {
	return container.Id
}

func (container *SContainer) GetGlobalId() string {
	return container.Id
}

func (container *SContainer) GetName() string {
	return container.Name
}

func (container *SContainer) GetStatus() string {
	return container.Status
}

func (container *SContainer) GetStartedAt() time.Time {
	return container.StartedAt
}

func (container *SContainer) GetLastFinishedAt() time.Time {
	return container.LastFinishedAt
}

func (container *SContainer) GetRestartCount() int {
	return container.RestartCount
}

func (container *SContainer) GetVolumentMounts() ([]cloudprovider.ICloudVolumeMount, error) {
	if container.Spec == nil {
		return []cloudprovider.ICloudVolumeMount{}, nil
	}
	ret := []cloudprovider.ICloudVolumeMount{}
	for id := range container.Spec.VolumeMounts {
		ret = append(ret, &SContainerVolumeMount{
			container.Spec.VolumeMounts[id],
		})
	}
	return ret, nil
}

func (container *SContainer) GetDevices() ([]cloudprovider.IContainerDevice, error) {
	if container.Spec == nil {
		return []cloudprovider.IContainerDevice{}, nil
	}
	ret := []cloudprovider.IContainerDevice{}
	for id := range container.Spec.Devices {
		ret = append(ret, &SContainerDevice{
			container.Spec.Devices[id],
		})
	}
	return ret, nil
}

type SContainerVolumeMount struct {
	*apis.ContainerVolumeMount
}

func (volumeMount *SContainerVolumeMount) GetName() string {
	return volumeMount.UniqueName
}

func (volumeMount *SContainerVolumeMount) IsReadOnly() bool {
	return volumeMount.ReadOnly
}

func (volumeMount *SContainerVolumeMount) GetType() string {
	return string(volumeMount.Type)
}

type SContainerDevice struct {
	*api.ContainerDevice
}

func (device *SContainerDevice) GetId() string {
	if device.IsolatedDevice != nil {
		return device.IsolatedDevice.Id
	}
	return ""
}

func (device *SContainerDevice) GetType() string {
	return string(device.Type)
}

func (container *SContainer) GetImage() string {
	if container.Spec == nil {
		return ""
	}
	return container.Spec.Image
}

func (container *SContainer) GetCommand() []string {
	if container.Spec == nil {
		return []string{}
	}
	return container.Spec.Command
}

func (container *SContainer) GetEnvs() []cloudprovider.SContainerEnv {
	if container.Spec == nil {
		return []cloudprovider.SContainerEnv{}
	}
	envs := make([]cloudprovider.SContainerEnv, 0)
	for _, env := range container.Spec.Envs {
		envs = append(envs, cloudprovider.SContainerEnv{
			Key:   env.Key,
			Value: env.Value,
		})
	}
	return envs
}

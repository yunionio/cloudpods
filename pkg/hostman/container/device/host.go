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

package device

import (
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/onecloud/pkg/apis"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
)

func init() {
	RegisterDriver(newHostDevice())
}

type hostDevice struct {
}

func newHostDevice() IDeviceDriver {
	return &hostDevice{}
}

func (h hostDevice) GetType() apis.ContainerDeviceType {
	return apis.CONTAINER_DEVICE_TYPE_HOST
}

func (h hostDevice) GetRuntimeDevices(input *hostapi.ContainerCreateInput, devs []*hostapi.ContainerDevice) ([]*runtimeapi.Device, error) {
	res := make([]*runtimeapi.Device, len(devs))
	for i, dev := range devs {
		res[i] = &runtimeapi.Device{
			ContainerPath: dev.ContainerPath,
			HostPath:      dev.Host.HostPath,
			Permissions:   dev.Permissions,
		}
	}

	return res, nil
}

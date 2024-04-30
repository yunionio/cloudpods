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

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
)

func init() {
	RegisterDriver(newIsolatedDevice())
}

type isolatedDevice struct{}

func newIsolatedDevice() IDeviceDriver {
	return &isolatedDevice{}
}

func (i isolatedDevice) GetType() apis.ContainerDeviceType {
	return apis.CONTAINER_DEVICE_TYPE_ISOLATED_DEVICE
}

func (i isolatedDevice) GetRuntimeDevices(input *hostapi.ContainerCreateInput, dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, error) {
	man, err := isolated_device.GetContainerDeviceManager(isolated_device.ContainerDeviceType(dev.IsolatedDevice.DeviceType))
	if err != nil {
		return nil, errors.Wrapf(err, "GetContainerDeviceManager by type %q", dev.Type)
	}
	ctrDevs, err := man.NewContainerDevices(input, dev)
	if err != nil {
		return nil, errors.Wrapf(err, "NewContainerDevices with %#v", dev)
	}
	return ctrDevs, nil
}

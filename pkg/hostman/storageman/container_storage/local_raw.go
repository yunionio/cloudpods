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

package container_storage

import (
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	losetupman "yunion.io/x/onecloud/pkg/util/losetup/manager"
)

func init() {
	isolated_device.RegisterContainerDeviceManager(newLocalLoopDiskManager())
}

type localLoopDiskManager struct {
}

func (l localLoopDiskManager) GetType() isolated_device.ContainerDeviceType {
	return api.CONTAINER_STORAGE_LOCAL_RAW
}

func (l localLoopDiskManager) NewDevices(dev *isolated_device.ContainerDevice) ([]isolated_device.IDevice, error) {
	return nil, errors.Errorf("%s storage doesn't support NewDevices", l.GetType())
}

func (l localLoopDiskManager) NewContainerDevices(_ *hostapi.ContainerCreateInput, input *hostapi.ContainerDevice) ([]*runtimeapi.Device, []*runtimeapi.Device, error) {
	dev := input.Disk
	disk, err := storageman.GetManager().GetDiskById(dev.Id)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "GetDiskById %s", dev.Id)
	}
	format, err := disk.GetFormat()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "get disk %s format", dev.Id)
	}
	if format != "raw" {
		return nil, nil, errors.Errorf("disk %s format isn't raw", dev.Id)
	}
	dPath := disk.GetPath()
	loDev, err := losetupman.AttachDevice(dPath, false)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to attach %s as loop device", dPath)
	}
	retDev := &runtimeapi.Device{
		ContainerPath: input.ContainerPath,
		HostPath:      loDev.Name,
		Permissions:   "rwm",
	}
	return []*runtimeapi.Device{retDev}, nil, nil
}

func (m *localLoopDiskManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	return nil, nil
}

func (m *localLoopDiskManager) GetContainerExtraConfigures(devs []*hostapi.ContainerDevice) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	return nil, nil
}

func newLocalLoopDiskManager() *localLoopDiskManager {
	return &localLoopDiskManager{}
}

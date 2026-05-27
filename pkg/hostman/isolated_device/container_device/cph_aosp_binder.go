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

package container_device

import (
	"fmt"
	"os"
	"path/filepath"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"yunion.io/x/onecloud/pkg/apis/compute"

	"yunion.io/x/pkg/errors"

	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

const (
	CPH_AOSP_BINDER_CONTROL_DEV_PATH = "/dev/binder-control"
	CPH_AOSP_BINDER_MODEL_NAME       = "CPH AOSP BINDER"
	CPH_AOSP_VENDOR_ID               = "0000"
	CPH_AOSP_DEVICE_ID               = "0000"
)

func init() {
	isolated_device.RegisterContainerDeviceManager(newCphAOSPBinderManager())
}

type cphAOSPBinderManager struct {
	controlDevicePath string
	controlName       string
	initialized       bool
}

func newCphAOSPBinderManager() *cphAOSPBinderManager {
	return &cphAOSPBinderManager{}
}

func (m *cphAOSPBinderManager) GetRegisterType() isolated_device.ContainerDeviceType {
	return isolated_device.ContainerDeviceTypeCphASOPBinder
}

func (m *cphAOSPBinderManager) GetDevType() string {
	return compute.BINDER_TYPE
}

func (m *cphAOSPBinderManager) GetSharingMode() string {
	return compute.DEVICE_SHARING_MODE_UNLIMITED
}

func (m *cphAOSPBinderManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	return nil, nil
}

func (m *cphAOSPBinderManager) NewDevices(dev *isolated_device.ContainerDevice) ([]isolated_device.IDevice, error) {
	if err := CheckVirtualNumber(dev); err != nil {
		return nil, err
	}
	if err := m.initialize(dev); err != nil {
		return nil, errors.Wrap(err, "initialize")
	}

	id := "aosp_binder"
	ndev := &isolated_device.PCIDevice{
		Addr:      m.controlName,
		VendorId:  CPH_AOSP_VENDOR_ID,
		DeviceId:  CPH_AOSP_DEVICE_ID,
		ModelName: CPH_AOSP_BINDER_MODEL_NAME,
	}
	devPath := fmt.Sprintf("/dev/%s", id)
	binderDev := &cphAOSPBinder{
		manager:     m,
		BaseDevice:  NewBaseDevice(ndev, compute.BINDER_TYPE, devPath, compute.DEVICE_SHARING_MODE_UNLIMITED, dev.VirtualNumber),
		ControlPath: m.controlDevicePath,
	}
	return []isolated_device.IDevice{binderDev}, nil
}

func (m *cphAOSPBinderManager) initialize(dev *isolated_device.ContainerDevice) error {
	if m.initialized {
		return errors.Errorf("cphAOSPBinderManager already initialized")
	}
	ctrlPath := CPH_AOSP_BINDER_CONTROL_DEV_PATH
	info, err := os.Stat(ctrlPath)
	if err != nil {
		return errors.Wrapf(err, "get status of path %s", ctrlPath)
	}
	m.controlDevicePath = ctrlPath
	m.controlName = info.Name()
	m.initialized = true
	return nil
}

func (m *cphAOSPBinderManager) NewContainerDevices(ctrInput *hostapi.ContainerCreateInput, input *hostapi.ContainerDevice) ([]*runtimeapi.Device, []*runtimeapi.Device, error) {
	dev := input.IsolatedDevice
	if err := m.ensureBinderDevice(ctrInput.Name, dev); err != nil {
		return nil, nil, errors.Wrap(err, "createBinderDevice")
	}
	binderFs := "/dev/binderfs"
	binderDev := func(devName string) string {
		return m.getBinderHostDevPath(ctrInput.Name, devName)
	}
	ctrDevs := []*runtimeapi.Device{
		{
			ContainerPath: filepath.Join(binderFs, "binder"),
			HostPath:      binderDev("binder"),
			Permissions:   "rwm",
		},
		{
			ContainerPath: filepath.Join(binderFs, "hwbinder"),
			HostPath:      binderDev("hwbinder"),
			Permissions:   "rwm",
		},
		{
			ContainerPath: filepath.Join(binderFs, "vndbinder"),
			HostPath:      binderDev("vndbinder"),
			Permissions:   "rwm",
		},
	}
	return ctrDevs, nil, nil
}

func (m *cphAOSPBinderManager) GetContainerExtraConfigures(devs []*hostapi.ContainerDevice) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	return nil, nil
}

func (m *cphAOSPBinderManager) ensureBinderDeviceOldWay(dev *hostapi.ContainerIsolatedDevice) error {
	if fileutils2.Exists(dev.Path) {
		return nil
	}
	binderBin := "/opt/yunion/bin/binder_device"
	baseName := filepath.Base(dev.Path)
	if err := procutils.NewRemoteCommandAsFarAsPossible(binderBin, CPH_AOSP_BINDER_CONTROL_DEV_PATH, baseName).Run(); err != nil {
		return errors.Wrapf(err, "call command: %s %s %s", binderBin, CPH_AOSP_BINDER_CONTROL_DEV_PATH, baseName)
	}
	return nil
}

func (m *cphAOSPBinderManager) getBinderHostDevPath(ctrName, devName string) string {
	binderFsPath := "/dev/binderfs"
	return filepath.Join(binderFsPath, ctrName, devName)
}

func (m *cphAOSPBinderManager) ensureBinderDevice(ctrName string, dev *hostapi.ContainerIsolatedDevice) error {
	binderBin := "/opt/yunion/bin/binder_devices_manager"
	binderDev := func(devName string) string {
		return m.getBinderHostDevPath(ctrName, devName)
	}
	if fileutils2.Exists(binderDev("binder")) && fileutils2.Exists(binderDev("vndbinder")) && fileutils2.Exists(binderDev("hwbinder")) {
		return nil
	}
	if out, err := procutils.NewRemoteCommandAsFarAsPossible(binderBin, ctrName).Output(); err != nil {
		return errors.Wrapf(err, "call command: %s %s, out: %s", binderBin, ctrName, out)
	}
	return nil
}

type cphAOSPBinder struct {
	manager *cphAOSPBinderManager
	*BaseDevice
	ControlPath string
}

func (dev *cphAOSPBinder) GetContainerDeviceManager() isolated_device.IContainerDeviceManager {
	return dev.manager
}

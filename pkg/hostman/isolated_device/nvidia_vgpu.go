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
	"io/ioutil"
	"path"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

type sNVIDIAVgpuDevice struct {
	pfDev   *PCIDevice
	cloudId string
	hostId  string
	guestId string
	devType string

	mdevId  string
	model   string
	profile map[string]string
}

func (dev *sNVIDIAVgpuDevice) String() string {
	return jsonutils.Marshal(dev).String()
}

func (dev *sNVIDIAVgpuDevice) IsInfinibandNic() bool {
	return false
}

func (dev *sNVIDIAVgpuDevice) GetCloudId() string {
	return dev.cloudId
}

func (dev *sNVIDIAVgpuDevice) GetHostId() string {
	return dev.hostId
}

func (dev *sNVIDIAVgpuDevice) SetHostId(hId string) {
	dev.hostId = hId
}

func (dev *sNVIDIAVgpuDevice) GetGuestId() string {
	return dev.guestId
}

func (dev *sNVIDIAVgpuDevice) GetWireId() string {
	return ""
}

func (dev *sNVIDIAVgpuDevice) GetOvsOffloadInterfaceName() string {
	return ""
}

func (dev *sNVIDIAVgpuDevice) GetVendorDeviceId() string {
	return dev.pfDev.GetVendorDeviceId()
}

func (dev *sNVIDIAVgpuDevice) GetAddr() string {
	return dev.pfDev.Addr
}

func (dev *sNVIDIAVgpuDevice) GetDeviceType() string {
	return dev.devType
}

func (dev *sNVIDIAVgpuDevice) GetModelName() string {
	modelName := dev.pfDev.ModelName
	if dev.pfDev.ModelName == "" {
		modelName = dev.pfDev.DeviceName
	}
	return modelName + "-" + dev.model
}

func (dev *sNVIDIAVgpuDevice) CustomProbe(idx int) error {
	return nil
}

func (dev *sNVIDIAVgpuDevice) GetDevicePath() string {
	return ""
}

func (dev *sNVIDIAVgpuDevice) SetDeviceInfo(info CloudDeviceInfo) {
	if len(info.Id) != 0 {
		dev.cloudId = info.Id
	}
	if len(info.GuestId) != 0 {
		dev.guestId = info.GuestId
	}
	if len(info.HostId) != 0 {
		dev.hostId = info.HostId
	}
	if len(info.DevType) != 0 {
		dev.devType = info.DevType
	}
}

func (dev *sNVIDIAVgpuDevice) GetNVIDIAVgpuProfile() map[string]string {
	return dev.profile
}

func (dev *sNVIDIAVgpuDevice) GetMdevId() string {
	return dev.mdevId
}

func (dev *sNVIDIAVgpuDevice) DetectByAddr() error {
	return nil
}

func (dev *sNVIDIAVgpuDevice) GetPassthroughOptions() map[string]string {
	return nil
}

func (dev *sNVIDIAVgpuDevice) GetPassthroughCmd(index int) string {
	return ""
}

func (dev *sNVIDIAVgpuDevice) GetIOMMUGroupDeviceCmd() string {
	return ""
}

func (dev *sNVIDIAVgpuDevice) GetIOMMUGroupRestAddrs() []string {
	return nil
}

func (dev *sNVIDIAVgpuDevice) GetPfName() string {
	return ""
}

func (dev *sNVIDIAVgpuDevice) GetVirtfn() int {
	return -1
}

func (dev *sNVIDIAVgpuDevice) GetNVMESizeMB() int {
	return -1
}

func (dev *sNVIDIAVgpuDevice) GetVGACmd() string {
	return ""
}

func (dev *sNVIDIAVgpuDevice) GetCPUCmd() string {
	return ""
}

func (dev *sNVIDIAVgpuDevice) GetQemuId() string {
	return "dev_" + dev.mdevId
}

func (dev *sNVIDIAVgpuDevice) GetHotPlugOptions(isolatedDev *desc.SGuestIsolatedDevice, guestDesc *desc.SGuestDesc) ([]*HotPlugOption, error) {
	ret := make([]*HotPlugOption, 0)

	var masterDevOpt *HotPlugOption
	for i := 0; i < len(isolatedDev.VfioDevs); i++ {
		sysfsdev := path.Join("/sys/bus/mdev/devices", isolatedDev.MdevId)
		opts := map[string]string{
			"sysfsdev": sysfsdev,
			"bus":      isolatedDev.VfioDevs[i].BusStr(),
			"addr":     isolatedDev.VfioDevs[i].SlotFunc(),
			"id":       isolatedDev.VfioDevs[i].Id,
		}
		if isolatedDev.VfioDevs[i].Multi != nil {
			if *isolatedDev.VfioDevs[i].Multi {
				opts["multifunction"] = "on"
			} else {
				opts["multifunction"] = "off"
			}
		}

		devOpt := &HotPlugOption{
			Device:  isolatedDev.VfioDevs[i].DevType,
			Options: opts,
		}
		if isolatedDev.VfioDevs[i].Function == 0 {
			masterDevOpt = devOpt
		} else {
			ret = append(ret, devOpt)
		}
	}
	// if PCI slot function 0 already assigned, qemu will reject hotplug function
	// so put function 0 at the enda
	if masterDevOpt == nil {
		return nil, errors.Errorf("GPU Device no function 0 found")
	}
	ret = append(ret, masterDevOpt)
	return ret, nil
}

func (dev *sNVIDIAVgpuDevice) GetHotUnplugOptions(isolatedDev *desc.SGuestIsolatedDevice) ([]*HotUnplugOption, error) {
	if len(isolatedDev.VfioDevs) == 0 {
		return nil, errors.Errorf("device %s no pci ids", isolatedDev.Id)
	}

	return []*HotUnplugOption{
		{
			Id: isolatedDev.VfioDevs[0].Id,
		},
	}, nil
}

// GetPCIEInfo implements IDevice.
func (dev *sNVIDIAVgpuDevice) GetPCIEInfo() *compute.IsolatedDevicePCIEInfo {
	return dev.pfDev.PCIEInfo
}

func NewNvidiaVgpuDevice(dev *PCIDevice, devType, mdevId, model string, profile map[string]string) *sNVIDIAVgpuDevice {
	return &sNVIDIAVgpuDevice{
		pfDev:   dev,
		devType: devType,
		mdevId:  mdevId,
		model:   model,
		profile: profile,
	}
}

func getNvidiaVGpus(gpuPF string) ([]*sNVIDIAVgpuDevice, error) {
	mdevDeviceDir := fmt.Sprintf("/sys/class/mdev_bus/0000:%s", gpuPF)
	if !fileutils2.Exists(mdevDeviceDir) {
		return nil, errors.Errorf("unknown device %s", gpuPF)
	}

	pfDev, err := detectPCIDevByAddrWithoutIOMMUGroup(gpuPF)
	if err != nil {
		return nil, errors.Wrap(err, "detect pf device")
	}
	// regutils.MatchUUID(self.HostId)
	files, err := ioutil.ReadDir(mdevDeviceDir)
	if err != nil {
		return nil, errors.Wrap(err, "read mdev device path")
	}
	nvidiaVgpus := make([]*sNVIDIAVgpuDevice, 0)
	for i := range files {
		if !regutils.MatchUUID(files[i].Name()) {
			continue
		}

		mdevPath := path.Join(mdevDeviceDir, files[i].Name())
		model, err := fileutils2.FileGetContents(path.Join(mdevPath, "mdev_type", "name"))
		if err != nil {
			return nil, errors.Wrap(err, "read file mdev_type/name")
		}
		model = strings.TrimSpace(model)
		// eg: num_heads=4, frl_config=60, framebuffer=1024M, max_resolution=5120x2880, max_instance=24
		vgpuProfile, err := fileutils2.FileGetContents(path.Join(mdevPath, "mdev_type", "description"))
		if err != nil {
			return nil, errors.Wrap(err, "read file mdev_type/description")
		}
		keys := []string{"num_heads", "frl_config", "framebuffer", "max_resolution", "max_instance"}
		profile := make(map[string]string)
		for _, key := range keys {
			keyWithValue := key + "="
			if strings.Contains(vgpuProfile, keyWithValue) {
				startIndex := strings.Index(vgpuProfile, keyWithValue)
				endIndex := startIndex + len(keyWithValue)
				value := strings.Split(vgpuProfile[endIndex:], ",")[0]
				profile[key] = strings.TrimSpace(value)
			}
		}
		mdev := NewNvidiaVgpuDevice(pfDev, compute.LEGACY_VGPU_TYPE, files[i].Name(), model, profile)
		nvidiaVgpus = append(nvidiaVgpus, mdev)
	}
	return nvidiaVgpus, nil
}

package container_device

import (
	"strconv"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	fileutils "yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func init() {
	isolated_device.RegisterContainerDeviceManager(newAMDGPUManager())
}

type amdGPUManager struct{}

func (m *amdGPUManager) GetRegisterType() isolated_device.ContainerDeviceType {
	return isolated_device.ContainerDeviceTypeAmdGpu
}

func (m *amdGPUManager) NewDevices(dev *isolated_device.ContainerDevice) ([]isolated_device.IDevice, error) {
	return nil, nil
}

func (m *amdGPUManager) NewContainerDevices(input *hostapi.ContainerCreateInput, dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, []*runtimeapi.Device, error) {
	hostPath := dev.IsolatedDevice.Path
	cDev := &runtimeapi.Device{
		ContainerPath: hostPath,
		HostPath:      hostPath,
		Permissions:   "rwm",
	}
	var retCommon []*runtimeapi.Device = nil
	if fileutils.Exists("/dev/kfd") {
		kdfDev := &runtimeapi.Device{
			ContainerPath: "/dev/kfd",
			HostPath:      "/dev/kfd",
			Permissions:   "rwm",
		}
		retCommon = []*runtimeapi.Device{kdfDev}
	}
	return []*runtimeapi.Device{cDev}, retCommon, nil
}

func (m *amdGPUManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	return getAMDGpus(m, api.DEVICE_SHARING_MODE_EXCLUSIVE)
}

func (m *amdGPUManager) GetContainerExtraConfigures(devs []*hostapi.ContainerDevice) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	return nil, nil
	//mounts := []*runtimeapi.Mount{}
	//if fileutils.Exists("/dev/kfd") {
	//	mounts = append(mounts, &runtimeapi.Mount{
	//		ContainerPath: "/dev/kfd",
	//		HostPath:      "/dev/kfd",
	//		Readonly:      false,
	//	})
	//}
	//_, err := procutils.RemoteStat("/opt/rocm-6.4.0")
	//if err == nil {
	//	mounts = append(mounts, &runtimeapi.Mount{
	//		ContainerPath: "/opt/rocm",
	//		HostPath:      "/opt/rocm-6.4.0",
	//		Readonly:      true,
	//	})
	//}
	//
	//_, err = procutils.RemoteStat("/opt/amdgpu")
	//if err == nil {
	//	mounts = append(mounts, &runtimeapi.Mount{
	//		ContainerPath: "/opt/amdgpu",
	//		HostPath:      "/opt/amdgpu",
	//		Readonly:      true,
	//	})
	//}

	//return nil, mounts
}

func newAMDGPUManager() *amdGPUManager {
	return &amdGPUManager{}
}

type amdGPU struct {
	manager isolated_device.IContainerDeviceManager

	*BaseDevice
	memorySize int
}

func (dev *amdGPU) GetMemorySize() int {
	return dev.memorySize
}

func (dev *amdGPU) GetContainerDeviceManager() isolated_device.IContainerDeviceManager {
	return dev.manager
}

/*
rocm-smi -i --showbus --showmeminfo vram  --json
{
  "card0": {
    "Device Name": "AMD Instinct MI210",
    "Device ID": "0x740f",
    "Device Rev": "0x02",
    "Subsystem ID": "0x0c34",
    "GUID": "63484",
    "PCI Bus": "0000:23:00.0",
    "VRAM Total Memory (B)": "68702699520",
    "VRAM Total Used Memory (B)": "10977280"
  }
}
*/

func getAMDGpus(m isolated_device.IContainerDeviceManager, sharingMode string) ([]isolated_device.IDevice, error) {
	devs := make([]isolated_device.IDevice, 0)
	out, err := procutils.NewRemoteCommandAsFarAsPossible("/opt/rocm/bin/rocm-smi", "-i", "--showbus", "--showmeminfo", "vram", "--json").Output()
	if err != nil {
		return nil, errors.Wrapf(err, "rocm-smi")
	}
	cards, err := jsonutils.Parse(out)
	if err != nil {
		return nil, errors.Wrapf(err, "parse rocm-smi outputs %s", out)
	}
	cardMap, err := cards.GetMap()
	if err != nil {
		return nil, errors.Wrapf(err, "get cards map")
	}
	pciRenderMap, err := getHygonRenderPathMap()
	if err != nil {
		return nil, errors.Wrapf(err, "get cards map")
	}

	for _, card := range cardMap {
		devName, _ := card.GetString("Device Name")
		pciAddr, _ := card.GetString("PCI Bus")
		memTotalStr, _ := card.GetString("VRAM Total Memory (B)")
		memTotalB, err := strconv.ParseInt(memTotalStr, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "parse card %s memTotalStr %s", card, memTotalStr)
		}
		memTotalMb := memTotalB / 1024 / 1024
		//guid, _ := card.GetString("GUID")
		pciOutput, err := isolated_device.GetPCIStrByAddr(pciAddr)
		if err != nil {
			return nil, errors.Wrapf(err, "GetPCIStrByAddr %s", pciAddr)
		}
		dev := isolated_device.NewPCIDevice2(pciOutput[0])

		devPath := pciRenderMap[dev.Addr]
		gpuDev := &amdGPU{
			manager:    m,
			BaseDevice: NewBaseDevice(dev, computeapi.GPU_TYPE, devPath, sharingMode, 1),
		}
		gpuDev.SetModelName(devName)
		gpuDev.memorySize = int(memTotalMb)

		devs = append(devs, gpuDev)
	}
	return devs, nil
}

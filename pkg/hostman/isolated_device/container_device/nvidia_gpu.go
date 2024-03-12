package container_device

import (
	"strings"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/host"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func init() {
	isolated_device.RegisterContainerDeviceManager(newNvidiaGPUManager())
}

type nvidiaGPUManager struct{}

func newNvidiaGPUManager() *nvidiaGPUManager {
	return &nvidiaGPUManager{}
}

func (m *nvidiaGPUManager) GetType() isolated_device.ContainerDeviceType {
	return isolated_device.ContainerDeviceTypeNVIDIAGPU
}

func (m *nvidiaGPUManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	return getNvidiaGPUs()
}

func (m *nvidiaGPUManager) NewDevices(dev *isolated_device.ContainerDevice) ([]isolated_device.IDevice, error) {
	return nil, nil
}

func (m *nvidiaGPUManager) NewContainerDevices(input *hostapi.ContainerCreateInput, dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, error) {
	return nil, nil
}

func (m *nvidiaGPUManager) GetContainerEnvs(devs []*host.ContainerDevice) []*runtimeapi.KeyValue {
	gpuIds := []string{}
	for _, dev := range devs {
		if dev.IsolatedDevice == nil {
			continue
		}
		if isolated_device.ContainerDeviceType(dev.IsolatedDevice.DeviceType) != isolated_device.ContainerDeviceTypeNVIDIAGPU {
			continue
		}
		gpuIds = append(gpuIds, dev.IsolatedDevice.Path)
	}
	if len(gpuIds) == 0 {
		return nil
	}

	return []*runtimeapi.KeyValue{
		{
			Key:   "NVIDIA_VISIBLE_DEVICES",
			Value: strings.Join(gpuIds, ","),
		},
		{
			Key:   "NVIDIA_DRIVER_CAPABILITIES",
			Value: "all",
		},
	}
}

type nvidiaGPU struct {
	*BaseDevice
}

func getNvidiaGPUs() ([]isolated_device.IDevice, error) {
	devs := make([]isolated_device.IDevice, 0)
	// nvidia-smi --query-gpu=gpu_uuid,gpu_name,gpu_bus_id --format=csv
	// uuid, name, pci.bus_id
	// GPU-bc1a3bb9-55cb-8c52-c374-4f8b4f388a20, NVIDIA A800-SXM4-80GB, 00000000:10:00.0
	out, err := procutils.NewRemoteCommandAsFarAsPossible("nvidia-smi", "--query-gpu=gpu_uuid,gpu_name,gpu_bus_id", "--format=csv").Output()
	if err != nil {
		return nil, errors.Wrap(err, "nvidia-smi")
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "uuid") {
			continue
		}
		segs := strings.Split(line, ",")
		if len(segs) != 3 {
			log.Errorf("unknown nvidia-smi out line %s", line)
			continue
		}
		gpuId, gpuName, gpuPciAddr := strings.TrimSpace(segs[0]), strings.TrimSpace(segs[1]), strings.TrimSpace(segs[2])
		pciOutput, err := isolated_device.GetPCIStrByAddr(gpuPciAddr)
		if err != nil {
			return nil, errors.Wrapf(err, "GetPCIStrByAddr %s", gpuPciAddr)
		}
		dev := isolated_device.NewPCIDevice2(pciOutput[0])
		gpuDev := &nvidiaGPU{
			BaseDevice: NewBaseDevice(dev, isolated_device.ContainerDeviceTypeNVIDIAGPU, gpuId),
		}
		gpuDev.SetModelName(gpuName)

		devs = append(devs, gpuDev)
	}
	if len(devs) == 0 {
		return nil, nil
	}
	return devs, nil
}

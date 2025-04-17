package container_device

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
)

func init() {
	isolated_device.RegisterContainerDeviceManager(newNvidiaGPUShareManager())
}

type nvidiaGPUShareManager struct {
	nvidiaGPUManager
}

func newNvidiaGPUShareManager() *nvidiaGPUShareManager {
	return &nvidiaGPUShareManager{}
}

func (m *nvidiaGPUShareManager) GetType() isolated_device.ContainerDeviceType {
	return isolated_device.ContainerDeviceTypeNvidiaGpuShare
}

func (m *nvidiaGPUShareManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	return nil, nil
}

func (m *nvidiaGPUShareManager) NewDevices(dev *isolated_device.ContainerDevice) ([]isolated_device.IDevice, error) {
	if !strings.HasPrefix(dev.Path, "/dev/dri/renderD") {
		return nil, errors.Errorf("device path %q doesn't start with /dev/dri/renderD", dev.Path)
	}
	if err := CheckVirtualNumber(dev); err != nil {
		return nil, err
	}

	gpuDevs := make([]isolated_device.IDevice, 0)
	for i := 0; i < dev.VirtualNumber; i++ {
		gpuDev, err := newNvidiaGpuShare(dev.Path, i)
		if err != nil {
			return nil, errors.Wrapf(err, "new CPH AMD GPU with index %d", i)
		}
		gpuDevs = append(gpuDevs, gpuDev)
	}
	return gpuDevs, nil
}

type nvidiaGpuShareDev struct {
	nvidiaGPU

	CardPath string
}

func (dev *nvidiaGpuShareDev) GetCardPath() string {
	return dev.CardPath
}

type nvidiaGpuUsage struct {
	*nvidiaGPU

	Used bool
}

var nvidiaGpuUsages map[string]*nvidiaGpuUsage = nil

func getNvidiaGpuUsage() (map[string]*nvidiaGpuUsage, error) {
	if nvidiaGpuUsages != nil {
		return nvidiaGpuUsages, nil
	}
	devs, err := getNvidiaGPUs()
	if err != nil {
		return nil, err
	}
	if len(devs) == 0 {
		return nil, nil
	}
	gpuUsages := map[string]*nvidiaGpuUsage{}
	for i := range devs {
		gpuUsages[devs[i].GetAddr()] = &nvidiaGpuUsage{
			nvidiaGPU: devs[i],
			Used:      false,
		}
	}
	nvidiaGpuUsages = gpuUsages
	return nvidiaGpuUsages, nil
}

func newNvidiaGpuShare(devPath string, index int) (*nvidiaGpuShareDev, error) {
	devUsages, err := getNvidiaGpuUsage()
	if err != nil {
		return nil, errors.Wrap(err, "getNvidiaGpuUsage")
	}

	dev, err := newPCIGPURenderBaseDevice(devPath, index, isolated_device.ContainerDeviceTypeNvidiaGpuShare)
	if err != nil {
		return nil, errors.Wrap(err, "new PCIGPURenderBaseDevice")
	}
	devAddr := dev.GetOriginAddr()
	cardPath := path.Join("/dev/dri/by-path", fmt.Sprintf("pci-0000:%s-card", devAddr))
	cardLinkPath, err := filepath.EvalSymlinks(cardPath)
	if err != nil {
		return nil, errors.Wrapf(err, "read link of %s", cardPath)
	}

	_, ok := devUsages[devAddr]
	if !ok {
		return nil, errors.Errorf("newNvidiaGpuShare dev addr not found %s", devAddr)
	}
	devUsages[devAddr].Used = true

	return &nvidiaGpuShareDev{
		nvidiaGPU: nvidiaGPU{
			BaseDevice: dev,
			memSize:    devUsages[devAddr].memSize,
			gpuIndex:   devUsages[devAddr].gpuIndex,
		},
		CardPath: cardLinkPath,
	}, nil
}

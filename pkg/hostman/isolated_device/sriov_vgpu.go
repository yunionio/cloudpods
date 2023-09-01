package isolated_device

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

type sSRIOVGpuDevice struct {
	*sSRIOVBaseDevice
}

func NewSRIOVGpuDevice(dev *PCIDevice, devType string) *sSRIOVGpuDevice {
	return &sSRIOVGpuDevice{
		sSRIOVBaseDevice: newSRIOVBaseDevice(dev, devType),
	}
}

func (dev *sSRIOVGpuDevice) GetPfName() string {
	return ""
}

func (dev *sSRIOVGpuDevice) GetVirtfn() int {
	return -1
}

func getSRIOVGpus(gpuPF string) ([]*sSRIOVGpuDevice, error) {
	devicePath := fmt.Sprintf("/sys/bus/pci/devices/0000:%s", gpuPF)
	if !fileutils2.Exists(devicePath) {
		return nil, errors.Errorf("unknown device %s", gpuPF)
	}
	files, err := ioutil.ReadDir(devicePath)
	if err != nil {
		return nil, errors.Wrap(err, "read device path")
	}
	sriovGPUs := make([]*sSRIOVGpuDevice, 0)
	for i := range files {
		if strings.HasPrefix(files[i].Name(), "virtfn") {
			_, err := strconv.Atoi(files[i].Name()[len("virtfn"):])
			if err != nil {
				return nil, err
			}
			vfPath, err := filepath.EvalSymlinks(path.Join(devicePath, files[i].Name()))
			if err != nil {
				return nil, err
			}
			vfBDF := path.Base(vfPath)
			vfDev, err := detectSRIOVDevice(vfBDF)
			if err != nil {
				return nil, err
			}
			sriovGPUs = append(sriovGPUs, NewSRIOVGpuDevice(vfDev, compute.SRIOV_VGPU_TYPE))
		}
	}
	return sriovGPUs, err
}

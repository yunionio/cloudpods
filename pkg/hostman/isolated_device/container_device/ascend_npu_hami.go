package container_device

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/hostman/options"
	fileutils "yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func init() {
	isolated_device.RegisterContainerDeviceManager(newAscendNPUHamiManager())
}

type ascendNPUHamiManager struct {
	*ascendNPUManager
}

func newAscendNPUHamiManager() *ascendNPUHamiManager {
	return &ascendNPUHamiManager{}
}

func (m *ascendNPUHamiManager) GetRegisterType() isolated_device.ContainerDeviceType {
	return isolated_device.ContainerDeviceTypeAscendNpuHami
}

func (m *ascendNPUHamiManager) GetContainerExtraConfigures(devs []*hostapi.ContainerDevice) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	npus := []string{}
	memoryLimit := ""
	smLimit := ""
	for _, dev := range devs {
		if dev.IsolatedDevice == nil {
			continue
		}
		iDev := hostinfo.Instance().IsolatedDeviceMan.GetDeviceByCloudId(dev.IsolatedDevice.Id)
		devMan := iDev.GetContainerDeviceManager()
		if _, ok := devMan.(*ascendNPUHamiManager); !ok {
			continue
		}
		idx, err := extractPartitionNumber(dev.IsolatedDevice.Path)
		if err != nil {
			log.Errorf("failed to extract partition number %s: %s", dev.IsolatedDevice.Path, err)
		}
		npus = append(npus, strconv.Itoa(idx))
		if memoryLimit == "" {
			memoryLimit = strconv.Itoa(dev.IsolatedDevice.MemoryLimit)
		}
		if smLimit == "" && dev.IsolatedDevice.SmUtilLimit > 0 {
			smLimit = fmt.Sprintf("%d", dev.IsolatedDevice.SmUtilLimit)
		}
	}
	if len(npus) == 0 {
		return nil, nil
	}

	if !fileutils.Exists(options.HostOptions.AscendNpuHamiShmPath) {
		err := os.MkdirAll(options.HostOptions.AscendNpuHamiShmPath, 0755)
		if err != nil {
			log.Errorf("failed to create shm dir %s: %s", options.HostOptions.AscendNpuHamiShmPath, err)
		}
	}
	retEnvs := []*runtimeapi.KeyValue{
		{
			Key:   "ASCEND_VISIBLE_DEVICES",
			Value: strings.Join(npus, ","),
		},
		{
			Key:   "LD_PRELOAD",
			Value: options.HostOptions.AscendNpuHamiLibvnpuPath,
		},
		{
			Key:   "NPU_MEM_QUOTA",
			Value: memoryLimit,
		},
		{
			Key:   "NPU_GLOBAL_SHM_PATH",
			Value: "/hami-shared-region/global_registry",
		},
	}
	if len(smLimit) > 0 {
		retEnvs = append(retEnvs, &runtimeapi.KeyValue{
			Key:   "NPU_PRIORITY",
			Value: smLimit,
		})
	}
	return retEnvs, []*runtimeapi.Mount{
		{
			ContainerPath: "/hami-shared-region",
			HostPath:      options.HostOptions.AscendNpuHamiShmPath,
			Readonly:      false,
		},
		{
			ContainerPath: options.HostOptions.AscendNpuHamiLibvnpuPath,
			HostPath:      options.HostOptions.AscendNpuHamiLibvnpuPath,
			Readonly:      true,
		},
	}
}

func (m *ascendNPUHamiManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	devs, err := getAscendNpus(m, computeapi.DEVICE_SHARING_MODE_HAMI)
	if err != nil {
		return nil, err
	}
	for i := range devs {
		devPath := devs[i].GetDevicePath()
		idx, err := extractPartitionNumber(devPath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract partition number %s", devPath)
		}
		out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", fmt.Sprintf("echo y | npu-smi set -t device-share -i %d -d 1", idx)).Output()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to set device-share %s: %s", devPath, out)
		}
	}
	return devs, nil
}

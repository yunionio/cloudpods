package container_storage

import (
	losetup "github.com/zexi/golosetup"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
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

func (l localLoopDiskManager) NewContainerDevices(_ *hostapi.ContainerCreateInput, input *hostapi.ContainerDevice) ([]*runtimeapi.Device, error) {
	dev := input.Disk
	disk, err := storageman.GetManager().GetDiskById(dev.Id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDiskById %s", dev.Id)
	}
	format, err := disk.GetFormat()
	if err != nil {
		return nil, errors.Wrapf(err, "get disk %s format", dev.Id)
	}
	if format != "raw" {
		return nil, errors.Errorf("disk %s format isn't raw", dev.Id)
	}
	dPath := disk.GetPath()
	loDev, err := losetup.AttachDevice(dPath, false)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to attach %s as loop device", dPath)
	}
	retDev := &runtimeapi.Device{
		ContainerPath: input.ContainerPath,
		HostPath:      loDev.Name,
		Permissions:   "rwm",
	}
	return []*runtimeapi.Device{retDev}, nil
}

func (m *localLoopDiskManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	return nil, nil
}

func (m *localLoopDiskManager) GetContainerEnvs(devs []*hostapi.ContainerDevice) []*runtimeapi.KeyValue {
	return nil
}

func newLocalLoopDiskManager() *localLoopDiskManager {
	return &localLoopDiskManager{}
}

package volume_mount

import (
	"fmt"
	"path/filepath"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/hostman/container/storage"
	container_storage "yunion.io/x/onecloud/pkg/hostman/container/storage"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func init() {
	RegisterDriver(newDisk())
}

type disk struct{}

func newDisk() IVolumeMount {
	return &disk{}
}

func (d disk) GetType() apis.ContainerVolumeMountType {
	return apis.CONTAINER_VOLUME_MOUNT_TYPE_DISK
}

func (d disk) GetRuntimeMountHostPath(pod IPodInfo, vm *apis.ContainerVolumeMount) (string, error) {
	diskInput := vm.Disk
	if diskInput == nil {
		return "", httperrors.NewNotEmptyError("disk is nil")
	}
	hostPath := filepath.Join(pod.GetVolumesDir(), diskInput.Id)
	if diskInput.SubDirectory != "" {
		return filepath.Join(hostPath, diskInput.SubDirectory), nil
	}
	if diskInput.StorageSizeFile != "" {
		return filepath.Join(hostPath, diskInput.StorageSizeFile), nil
	}
	return hostPath, nil
}

func (d disk) getPodDisk(pod IPodInfo, vm *apis.ContainerVolumeMount) (storageman.IDisk, *desc.SGuestDisk, error) {
	var disk *desc.SGuestDisk = nil
	disks := pod.GetDisks()
	volDisk := vm.Disk
	if volDisk.Id == "" {
		return nil, nil, errors.Errorf("volume mount disk id is empty")
	}
	if volDisk.Id != "" {
		for _, gd := range disks {
			if gd.DiskId == volDisk.Id {
				disk = gd
				break
			}
		}
	}
	if disk == nil {
		return nil, nil, errors.Wrapf(errors.ErrNotFound, "not found disk by id %s", volDisk.Id)
	}
	iDisk, err := storageman.GetManager().GetDiskById(disk.DiskId)
	if err != nil {
		return nil, disk, errors.Wrapf(err, "GetDiskById %s", disk.Path)
	}
	return iDisk, disk, nil
}

func (d disk) getDiskStorageDriver(pod IPodInfo, vm *apis.ContainerVolumeMount) (storage.IContainerStorage, error) {
	iDisk, _, err := d.getPodDisk(pod, vm)
	if err != nil {
		return nil, errors.Wrap(err, "get pod disk interface")
	}
	drv, err := iDisk.GetContainerStorageDriver()
	if err != nil {
		return nil, errors.Wrap(err, "GetContainerStorageDriver")
	}
	return drv, nil
}

func (d disk) Mount(pod IPodInfo, vm *apis.ContainerVolumeMount) error {
	iDisk, gd, err := d.getPodDisk(pod, vm)
	if err != nil {
		return errors.Wrap(err, "get pod disk interface")
	}
	drv, err := iDisk.GetContainerStorageDriver()
	if err != nil {
		return errors.Wrap(err, "get disk storage driver")
	}
	devPath, isConnected, err := drv.CheckConnect(iDisk.GetPath())
	if err != nil {
		return errors.Wrapf(err, "CheckConnect %s", iDisk.GetPath())
	}
	log.Infof("=======check connect: %q %q %v", iDisk.GetPath(), devPath, isConnected)
	if !isConnected {
		devPath, err = drv.ConnectDisk(iDisk.GetPath())
		if err != nil {
			return errors.Wrapf(err, "ConnectDisk %s", iDisk.GetPath())
		}
	}
	mntPoint := pod.GetDiskMountPoint(iDisk)
	if err := container_storage.Mount(devPath, mntPoint, gd.Fs); err != nil {
		return errors.Wrapf(err, "mount %s to %s", devPath, mntPoint)
	}
	vmDisk := vm.Disk
	if vmDisk.SubDirectory != "" {
		out, err := procutils.NewRemoteCommandAsFarAsPossible("mkdir", "-p", filepath.Join(mntPoint, vmDisk.SubDirectory)).Output()
		if err != nil {
			return errors.Wrapf(err, "make sub_directory %s inside %s: %s", vmDisk.SubDirectory, mntPoint, out)
		}
	}
	if vmDisk.StorageSizeFile != "" {
		if err := d.createStorageSizeFile(iDisk, mntPoint, vmDisk); err != nil {
			return errors.Wrapf(err, "create storage file %s inside %s", vmDisk.StorageSizeFile, mntPoint)
		}
	}
	return nil
}

func (d disk) createStorageSizeFile(iDisk storageman.IDisk, mntPoint string, input *apis.ContainerVolumeMountDisk) error {
	desc := iDisk.GetDiskDesc()
	diskSizeMB, err := desc.Int("disk_size")
	if err != nil {
		return errors.Wrapf(err, "get disk_size from %s", desc.String())
	}
	sp := filepath.Join(mntPoint, input.StorageSizeFile)
	sizeBytes := diskSizeMB * 1024
	out, err := procutils.NewRemoteCommandAsFarAsPossible("bash", "-c", fmt.Sprintf("echo %d > %s", sizeBytes, sp)).Output()
	if err != nil {
		return errors.Wrapf(err, "write %d to %s: %s", sizeBytes, sp, out)
	}
	return nil
}

func (d disk) Unmount(pod IPodInfo, vm *apis.ContainerVolumeMount) error {
	iDisk, _, err := d.getPodDisk(pod, vm)
	if err != nil {
		return errors.Wrap(err, "get pod disk interface")
	}
	drv, err := iDisk.GetContainerStorageDriver()
	if err != nil {
		return errors.Wrap(err, "get disk storage driver")
	}
	mntPoint := pod.GetDiskMountPoint(iDisk)
	if err := container_storage.Unmount(mntPoint); err != nil {
		return errors.Wrapf(err, "unmount %s", mntPoint)
	}
	_, isConnected, err := drv.CheckConnect(iDisk.GetPath())
	if err != nil {
		return errors.Wrapf(err, "CheckConnect %s", iDisk.GetPath())
	}
	if isConnected {
		if err := drv.DisconnectDisk(iDisk.GetPath(), mntPoint); err != nil {
			return errors.Wrapf(err, "DisconnectDisk %s %s", iDisk.GetPath(), mntPoint)
		}
	}
	return nil
}

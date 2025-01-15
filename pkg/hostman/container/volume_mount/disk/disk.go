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

package disk

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/hostman/container/storage"
	container_storage "yunion.io/x/onecloud/pkg/hostman/container/storage"
	"yunion.io/x/onecloud/pkg/hostman/container/volume_mount"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func init() {
	volume_mount.RegisterDriver(newDisk())
}

type IVolumeMountDisk interface {
	volume_mount.IUsageVolumeMount

	MountPostOverlays(pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, ovs []*apis.ContainerVolumeMountDiskPostOverlay) error
	UnmountPostOverlays(pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, ovs []*apis.ContainerVolumeMountDiskPostOverlay, useLazy bool, clearLayers bool) error

	GetHostDiskRootPath(pod volume_mount.IPodInfo, vm *hostapi.ContainerVolumeMount) (string, error)

	GetPostOverlayRootWorkDir(pod volume_mount.IPodInfo, vm *hostapi.ContainerVolumeMount, ctrId string) (string, error)
	GetPostOverlayRootUpperDir(pod volume_mount.IPodInfo, vm *hostapi.ContainerVolumeMount, ctrId string) (string, error)
}

type disk struct {
	overlayDrivers map[apis.ContainerDiskOverlayType]iDiskOverlay
}

func newDisk() IVolumeMountDisk {
	return &disk{
		overlayDrivers: map[apis.ContainerDiskOverlayType]iDiskOverlay{
			apis.CONTAINER_DISK_OVERLAY_TYPE_DIRECTORY:  newDiskOverlayDir(),
			apis.CONTAINER_DISK_OVERLAY_TYPE_DISK_IMAGE: newDiskOverlayImage(),
		},
	}
}

func (d disk) GetType() apis.ContainerVolumeMountType {
	return apis.CONTAINER_VOLUME_MOUNT_TYPE_DISK
}

func (d disk) GetHostDiskRootPath(pod volume_mount.IPodInfo, vm *hostapi.ContainerVolumeMount) (string, error) {
	diskInput := vm.Disk
	if diskInput == nil {
		return "", httperrors.NewNotEmptyError("disk is nil")
	}
	hostPath := filepath.Join(pod.GetVolumesDir(), diskInput.Id)
	return hostPath, nil
}

func (d disk) getRuntimeMountHostPath(pod volume_mount.IPodInfo, vm *hostapi.ContainerVolumeMount) (string, error) {
	hostPath, err := d.GetHostDiskRootPath(pod, vm)
	if err != nil {
		return "", errors.Wrap(err, "get host disk root path")
	}
	diskInput := vm.Disk
	if diskInput.SubDirectory != "" {
		return filepath.Join(hostPath, diskInput.SubDirectory), nil
	}
	if diskInput.StorageSizeFile != "" {
		return filepath.Join(hostPath, diskInput.StorageSizeFile), nil
	}
	return hostPath, nil
}

func (d disk) GetRuntimeMountHostPath(pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) (string, error) {
	hostPath, err := d.getRuntimeMountHostPath(pod, vm)
	if err != nil {
		return "", errors.Wrap(err, "get runtime mount host_path")
	}
	overlay := vm.Disk.Overlay
	if overlay == nil && vm.Disk.TemplateId == "" {
		return hostPath, nil
	}
	return d.getOverlayMergedDir(pod, ctrId, vm, hostPath), nil
}

func (d disk) getPodDisk(pod volume_mount.IPodInfo, vm *hostapi.ContainerVolumeMount) (storageman.IDisk, *desc.SGuestDisk, error) {
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

func (d disk) getDiskStorageDriver(pod volume_mount.IPodInfo, vm *hostapi.ContainerVolumeMount) (storage.IContainerStorage, error) {
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

func (d disk) setDirCaseInsensitive(dir string) error {
	out, err := procutils.NewRemoteCommandAsFarAsPossible("chattr", "+F", dir).Output()
	if err != nil {
		return errors.Wrapf(err, "enable %q case_insensitive: %s", dir, out)
	}
	return nil
}

func (d disk) newPostOverlay() iDiskPostOverlay {
	return newDiskPostOverlay(d)
}

func (d disk) connectDisk(iDisk storageman.IDisk) (string, bool, error) {
	drv, err := iDisk.GetContainerStorageDriver()
	if err != nil {
		return "", false, errors.Wrap(err, "get disk storage driver")
	}
	devPath, isConnected, err := drv.CheckConnect(iDisk.GetPath())
	if err != nil {
		return "", false, errors.Wrapf(err, "CheckConnect %s", iDisk.GetPath())
	}
	if !isConnected {
		devPath, err = drv.ConnectDisk(iDisk.GetPath())
		if err != nil {
			return "", false, errors.Wrapf(err, "ConnectDisk %s", iDisk.GetPath())
		}
	}
	return devPath, isConnected, nil
}

func (d disk) mountDisk(devPath string, mntPoint string, fs string) error {
	if err := container_storage.Mount(devPath, mntPoint, fs); err != nil {
		return errors.Wrapf(err, "mount %s to %s", devPath, mntPoint)
	}
	return nil
}

func (d disk) connectDiskAndMount(drv container_storage.IContainerStorage, pod volume_mount.IPodInfo, iDisk storageman.IDisk, fs string) (string, error) {
	devPath, isConnected, err := d.connectDisk(iDisk)
	if err != nil {
		return "", errors.Wrap(err, "connect disk")
	}
	mntPoint := pod.GetDiskMountPoint(iDisk)
	mountErrs := []error{}
	if err := d.mountDisk(devPath, mntPoint, fs); err != nil {
		mountErrs = append(mountErrs, err)
		if isConnected && strings.Contains(err.Error(), fmt.Sprintf("%s already mounted or mount point busy.", devPath)) {
			// disconnect disk and mount agin
			if err := drv.DisconnectDisk(iDisk.GetPath(), mntPoint); err != nil {
				mountErrs = append(mountErrs, errors.Wrapf(err, "disconnect disk cause of mount point busy"))
				return mntPoint, errors.NewAggregate(mountErrs)
			}
			devPath, _, err = d.connectDisk(iDisk)
			if err != nil {
				return mntPoint, errors.Wrap(err, "connect disk after disconnect")
			}
			if err := d.mountDisk(devPath, mntPoint, fs); err != nil {
				mountErrs = append(mountErrs, errors.Wrapf(err, "mount disk after reconnect"))
				return mntPoint, errors.NewAggregate(mountErrs)
			}
			return mntPoint, nil
		}
		return mntPoint, errors.Wrapf(err, "mount %s to %s", devPath, mntPoint)
	}
	return mntPoint, nil
}

func (d disk) Mount(pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error {
	iDisk, gd, err := d.getPodDisk(pod, vm)
	if err != nil {
		return errors.Wrap(err, "get pod disk interface")
	}
	drv, err := iDisk.GetContainerStorageDriver()
	if err != nil {
		return errors.Wrap(err, "get disk storage driver")
	}
	mntPoint, err := d.connectDiskAndMount(drv, pod, iDisk, gd.Fs)
	if err != nil {
		return errors.Wrap(err, "connect disk and mount disk")
	}

	vmDisk := vm.Disk
	if vmDisk.SubDirectory != "" {
		subDir := filepath.Join(mntPoint, vmDisk.SubDirectory)
		if err := volume_mount.EnsureDir(subDir); err != nil {
			return errors.Wrapf(err, "make sub_directory %s inside %s", vmDisk.SubDirectory, mntPoint)
		}
		for _, cd := range vmDisk.CaseInsensitivePaths {
			cdp := filepath.Join(subDir, cd)
			if err := volume_mount.EnsureDir(cdp); err != nil {
				return errors.Wrapf(err, "make %s inside %s", cdp, vmDisk.SubDirectory)
			}
			if err := d.setDirCaseInsensitive(cdp); err != nil {
				return errors.Wrapf(err, "enable case_insensitive %s", cdp)
			}
		}
	} else {
		if len(vmDisk.CaseInsensitivePaths) != 0 {
			return errors.Errorf("only sub_directory can use case_insensitive")
		}
	}
	if vmDisk.StorageSizeFile != "" {
		if err := d.createStorageSizeFile(iDisk, mntPoint, vmDisk); err != nil {
			return errors.Wrapf(err, "create storage file %s inside %s", vmDisk.StorageSizeFile, mntPoint)
		}
	}
	if vmDisk.Overlay != nil {
		if err := d.mountOverlay(pod, ctrId, vm); err != nil {
			return errors.Wrapf(err, "mount container %s overlay dir: %#v", ctrId, vmDisk.Overlay)
		}
	}
	if len(vmDisk.PostOverlay) != 0 {
		if err := d.MountPostOverlays(pod, ctrId, vm, vmDisk.PostOverlay); err != nil {
			return errors.Wrap(err, "mount post overlay dirs")
		}
	}
	return nil
}

func (d disk) createStorageSizeFile(iDisk storageman.IDisk, mntPoint string, input *hostapi.ContainerVolumeMountDisk) error {
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

func (d disk) Unmount(pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error {
	iDisk, _, err := d.getPodDisk(pod, vm)
	if err != nil {
		return errors.Wrap(err, "get pod disk interface")
	}
	drv, err := iDisk.GetContainerStorageDriver()
	if err != nil {
		return errors.Wrap(err, "get disk storage driver")
	}
	if len(vm.Disk.PostOverlay) != 0 {
		if err := d.UnmountPostOverlays(pod, ctrId, vm, vm.Disk.PostOverlay, false, false); err != nil {
			return errors.Wrap(err, "umount post overlay dirs")
		}
	}
	if vm.Disk.Overlay != nil {
		if err := d.unmoutOverlay(pod, ctrId, vm); err != nil {
			return errors.Wrapf(err, "umount overlay")
		}
	}
	mntPoint := pod.GetDiskMountPoint(iDisk)
	if err := container_storage.UnmountWithSubDirs(mntPoint); err != nil {
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
	if err := volume_mount.RemoveDir(mntPoint); err != nil {
		return errors.Wrapf(err, "remove dir %s", mntPoint)
	}
	return nil
}

func (d disk) getOverlayDriver(ov *apis.ContainerVolumeMountDiskOverlay) iDiskOverlay {
	return d.overlayDrivers[ov.GetType()]
}

func (d disk) unmoutOverlay(pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error {
	return d.getOverlayDriver(vm.Disk.Overlay).unmount(d, pod, ctrId, vm)
}

func (d disk) mountOverlay(pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error {
	return d.getOverlayDriver(vm.Disk.Overlay).mount(d, pod, ctrId, vm)
}

func (d disk) getCachedImageDir(ctx context.Context, pod volume_mount.IPodInfo, imgId string) (string, error) {
	input := computeapi.CacheImageInput{
		ImageId:              imgId,
		Format:               imageapi.IMAGE_DISK_FORMAT_TGZ,
		SkipChecksumIfExists: true,
	}
	cachedImgMan := storageman.GetManager().LocalStorageImagecacheManager
	logPrefx := fmt.Sprintf("pod %s, image %s", pod.GetName(), imgId)
	log.Infof("%s try to accuire image...", logPrefx)
	cachedImg, err := cachedImgMan.AcquireImage(ctx, input, nil)
	if err != nil {
		return "", errors.Wrapf(err, "Get cache image %s", imgId)
	}
	defer cachedImgMan.ReleaseImage(ctx, imgId)
	log.Infof("%s try to get access directory", logPrefx)
	cachedImageDir, err := cachedImg.GetAccessDirectory()
	if err != nil {
		return "", errors.Wrapf(err, "GetAccessDirectory of cached image %s", cachedImg.GetPath())
	}
	log.Infof("%s got cached image dir %s", logPrefx, cachedImageDir)
	return cachedImageDir, nil
}

func (d disk) doTemplateOverlayAction(
	ctx context.Context,
	pod volume_mount.IPodInfo, ctrId string,
	vm *hostapi.ContainerVolumeMount,
	ovAction func(d disk, pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error) error {
	templateId := vm.Disk.TemplateId
	cachedImageDir, err := d.getCachedImageDir(ctx, pod, templateId)
	if err != nil {
		return errors.Wrap(err, "get cached image dir")
	}
	vm.Disk.Overlay = &apis.ContainerVolumeMountDiskOverlay{
		LowerDir: []string{cachedImageDir},
	}
	if err := ovAction(d, pod, ctrId, vm); err != nil {
		return errors.Wrapf(err, "overlay dir %s", cachedImageDir)
	}
	return nil
}

func (d disk) InjectUsageTags(usage *volume_mount.ContainerVolumeMountUsage, vol *hostapi.ContainerVolumeMount) {
	usage.Tags["disk_id"] = vol.Disk.Id
}

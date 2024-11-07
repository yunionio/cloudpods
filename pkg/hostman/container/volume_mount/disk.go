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

package volume_mount

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
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

type iDiskOverlay interface {
	mount(d disk, pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error
	unmount(d disk, pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error
}

type disk struct {
	overlayDrivers map[apis.ContainerDiskOverlayType]iDiskOverlay
}

func newDisk() IVolumeMount {
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

func (d disk) getRuntimeMountHostPath(pod IPodInfo, vm *hostapi.ContainerVolumeMount) (string, error) {
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

func (d disk) GetRuntimeMountHostPath(pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) (string, error) {
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

func (d disk) getPodDisk(pod IPodInfo, vm *hostapi.ContainerVolumeMount) (storageman.IDisk, *desc.SGuestDisk, error) {
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

func (d disk) getDiskStorageDriver(pod IPodInfo, vm *hostapi.ContainerVolumeMount) (storage.IContainerStorage, error) {
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

func (d disk) getOverlayDir(pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, upperDir string, suffix string) string {
	return filepath.Join(pod.GetVolumesOverlayDir(), vm.Disk.Id, ctrId, fmt.Sprintf("%s-%s", filepath.Base(upperDir), suffix))
}

func (d disk) getOverlayWorkDir(upperDir string) string {
	return fmt.Sprintf("%s-work", upperDir)
}

func (d disk) getOverlayMergedDir(pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, upperDir string) string {
	return d.getOverlayDir(pod, ctrId, vm, upperDir, "merged")
}

func (d disk) setDirCaseInsensitive(dir string) error {
	out, err := procutils.NewRemoteCommandAsFarAsPossible("chattr", "+F", dir).Output()
	if err != nil {
		return errors.Wrapf(err, "enable %q case_insensitive: %s", dir, out)
	}
	return nil
}

func (d disk) Mount(pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error {
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
		subDir := filepath.Join(mntPoint, vmDisk.SubDirectory)
		out, err := procutils.NewRemoteCommandAsFarAsPossible("mkdir", "-p", subDir).Output()
		if err != nil {
			return errors.Wrapf(err, "make sub_directory %s inside %s: %s", vmDisk.SubDirectory, mntPoint, out)
		}
		for _, cd := range vmDisk.CaseInsensitivePaths {
			cdp := filepath.Join(subDir, cd)
			out, err := procutils.NewRemoteCommandAsFarAsPossible("mkdir", "-p", cdp).Output()
			if err != nil {
				return errors.Wrapf(err, "make %s inside %s: %s", cdp, vmDisk.SubDirectory, out)
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

func (d disk) Unmount(pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error {
	iDisk, _, err := d.getPodDisk(pod, vm)
	if err != nil {
		return errors.Wrap(err, "get pod disk interface")
	}
	drv, err := iDisk.GetContainerStorageDriver()
	if err != nil {
		return errors.Wrap(err, "get disk storage driver")
	}
	if vm.Disk.Overlay != nil {
		if err := d.unmoutOverlay(pod, ctrId, vm); err != nil {
			return errors.Wrapf(err, "umount overlay")
		}
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

func (d disk) getOverlayDriver(ov *apis.ContainerVolumeMountDiskOverlay) iDiskOverlay {
	return d.overlayDrivers[ov.GetType()]
}

func (d disk) unmoutOverlay(pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error {
	return d.getOverlayDriver(vm.Disk.Overlay).unmount(d, pod, ctrId, vm)
}

func (d disk) mountOverlay(pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error {
	return d.getOverlayDriver(vm.Disk.Overlay).mount(d, pod, ctrId, vm)
}

func (d disk) doTemplateOverlayAction(
	ctx context.Context,
	pod IPodInfo, ctrId string,
	vm *hostapi.ContainerVolumeMount,
	ovAction func(d disk, pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error) error {
	templateId := vm.Disk.TemplateId
	input := computeapi.CacheImageInput{
		ImageId:              templateId,
		SkipChecksumIfExists: true,
	}
	cachedImgMan := storageman.GetManager().LocalStorageImagecacheManager
	cachedImg, err := cachedImgMan.AcquireImage(ctx, input, nil)
	if err != nil {
		return errors.Wrapf(err, "Get cache image %s", templateId)
	}
	defer cachedImgMan.ReleaseImage(ctx, templateId)
	cachedImageDir, err := cachedImg.GetAccessDirectory()
	if err != nil {
		return errors.Wrapf(err, "GetAccessDirectory of cached image %s", cachedImg.GetPath())
	}
	vm.Disk.Overlay = &apis.ContainerVolumeMountDiskOverlay{
		LowerDir: []string{cachedImageDir},
	}
	if err := ovAction(d, pod, ctrId, vm); err != nil {
		return errors.Wrapf(err, "overlay dir %s", cachedImageDir)
	}
	return nil
}

func (d disk) InjectUsageTags(usage *ContainerVolumeMountUsage, vol *hostapi.ContainerVolumeMount) {
	usage.Tags["disk_id"] = vol.Disk.Id
}

type diskOverlayDir struct{}

func newDiskOverlayDir() iDiskOverlay {
	return &diskOverlayDir{}
}

func (dod diskOverlayDir) mount(d disk, pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error {
	vmDisk := vm.Disk
	lowerDir := vmDisk.Overlay.LowerDir
	upperDir, err := d.getRuntimeMountHostPath(pod, vm)
	if err != nil {
		return errors.Wrap(err, "getRuntimeMountHostPath")
	}
	workDir := d.getOverlayWorkDir(upperDir)
	mergedDir := d.getOverlayMergedDir(pod, ctrId, vm, upperDir)
	for _, dir := range []string{workDir, mergedDir} {
		out, err := procutils.NewRemoteCommandAsFarAsPossible("mkdir", "-p", dir).Output()
		if err != nil {
			return errors.Wrapf(err, "make directory %s: %s", dir, out)
		}
	}

	overlayArgs := []string{"-t", "overlay", "overlay", "-o", fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", strings.Join(lowerDir, ":"), upperDir, workDir), mergedDir}
	if out, err := procutils.NewRemoteCommandAsFarAsPossible("mount", overlayArgs...).Output(); err != nil {
		return errors.Wrapf(err, "mount %v: %s", overlayArgs, out)
	}

	return nil
}

func (dod diskOverlayDir) unmount(d disk, pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error {
	upperDir, err := d.getRuntimeMountHostPath(pod, vm)
	if err != nil {
		return errors.Wrap(err, "getRuntimeMountHostPath")
	}
	overlayDir := d.getOverlayMergedDir(pod, ctrId, vm, upperDir)
	return container_storage.Unmount(overlayDir)
}

type diskOverlayImage struct{}

func newDiskOverlayImage() iDiskOverlay {
	return &diskOverlayImage{}
}

func (di diskOverlayImage) mount(d disk, pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error {
	if err := d.doTemplateOverlayAction(context.Background(), pod, ctrId, vm, newDiskOverlayDir().mount); err != nil {
		return errors.Wrapf(err, "mount template overlay")
	}
	return nil
}

func (di diskOverlayImage) unmount(d disk, pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error {
	if err := d.doTemplateOverlayAction(context.Background(), pod, ctrId, vm, newDiskOverlayDir().unmount); err != nil {
		return errors.Wrapf(err, "unmount template overlay")
	}
	return nil
}

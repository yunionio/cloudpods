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
	"path/filepath"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/container/volume_mount"
	"yunion.io/x/onecloud/pkg/util/mountutils"
)

const (
	POST_OVERLAY_PREFIX_WORK_DIR  = "_post_overlay_work_"
	POST_OVERLAY_PREFIX_UPPER_DIR = "_post_overlay_upper_"
)

func (d disk) MountPostOverlays(pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, ovs []*apis.ContainerVolumeMountDiskPostOverlay) error {
	return d.newPostOverlay().mountPostOverlays(pod, ctrId, vm, ovs)
}

func (d disk) UnmountPostOverlays(pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, ovs []*apis.ContainerVolumeMountDiskPostOverlay, useLazy bool, clearLayers bool) error {
	return d.newPostOverlay().unmountPostOverlays(pod, ctrId, vm, ovs, useLazy, clearLayers)
}

func (d disk) getPostOverlayRootPrefixDir(prefixDir string, pod volume_mount.IPodInfo, vm *hostapi.ContainerVolumeMount, ctrId string) (string, error) {
	hostPath, err := d.GetHostDiskRootPath(pod, vm)
	if err != nil {
		return "", errors.Wrap(err, "get host disk root path")
	}
	uniqDir := ctrId
	if vm.UniqueName != "" {
		uniqDir = vm.UniqueName
	}
	return filepath.Join(hostPath, prefixDir, uniqDir), nil
}

func (d disk) GetPostOverlayRootWorkDir(pod volume_mount.IPodInfo, vm *hostapi.ContainerVolumeMount, ctrId string) (string, error) {
	return d.getPostOverlayRootPrefixDir(POST_OVERLAY_PREFIX_WORK_DIR, pod, vm, ctrId)
}

func (d disk) GetPostOverlayRootUpperDir(pod volume_mount.IPodInfo, vm *hostapi.ContainerVolumeMount, ctrId string) (string, error) {
	return d.getPostOverlayRootPrefixDir(POST_OVERLAY_PREFIX_UPPER_DIR, pod, vm, ctrId)
}

type iDiskPostOverlay interface {
	mountPostOverlays(pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, ovs []*apis.ContainerVolumeMountDiskPostOverlay) error
	unmountPostOverlays(pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, ovs []*apis.ContainerVolumeMountDiskPostOverlay, useLazy bool, clearLayers bool) error
}

type diskPostOverlay struct {
	disk disk
}

func newDiskPostOverlay(d disk) iDiskPostOverlay {
	return &diskPostOverlay{
		disk: d,
	}
}

func (d diskPostOverlay) mountPostOverlays(pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, ovs []*apis.ContainerVolumeMountDiskPostOverlay) error {
	for _, ov := range ovs {
		if err := d.mountPostOverlay(pod, ctrId, vm, ov); err != nil {
			return errors.Wrapf(err, "mount container %s post overlay dir: %#v", ctrId, ov)
		}
	}
	return nil
}

func (d diskPostOverlay) unmountPostOverlays(pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, ovs []*apis.ContainerVolumeMountDiskPostOverlay, useLazy bool, clearLayers bool) error {
	for _, ov := range ovs {
		if err := d.unmountPostOverlay(pod, ctrId, vm, ov, useLazy, clearLayers); err != nil {
			return errors.Wrapf(err, "unmount container %s post overlay dir: %#v", ctrId, ov)
		}
	}
	return nil
}

func (d diskPostOverlay) getPostOverlayDirWithPrefix(
	prefixDir string,
	pod volume_mount.IPodInfo, ctrId string,
	vm *hostapi.ContainerVolumeMount,
	ov *apis.ContainerVolumeMountDiskPostOverlay,
	ensure bool,
) (string, error) {
	rootPath, err := d.disk.getPostOverlayRootPrefixDir(prefixDir, pod, vm, ctrId)
	if err != nil {
		return "", errors.Wrap(err, "get post overlay root path")
	}

	workDir := filepath.Join(rootPath, ov.ContainerTargetDir)
	if ensure {
		if err := volume_mount.EnsureDir(workDir); err != nil {
			return "", errors.Wrapf(err, "make %s", workDir)
		}
	}
	return workDir, nil
}

func (d diskPostOverlay) getPostOverlayWorkDir(
	pod volume_mount.IPodInfo, ctrId string,
	vm *hostapi.ContainerVolumeMount,
	ov *apis.ContainerVolumeMountDiskPostOverlay,
	ensure bool,
) (string, error) {
	return d.getPostOverlayDirWithPrefix(POST_OVERLAY_PREFIX_WORK_DIR, pod, ctrId, vm, ov, ensure)
}

func (d diskPostOverlay) getPostOverlayUpperDir(
	pod volume_mount.IPodInfo, ctrId string,
	vm *hostapi.ContainerVolumeMount,
	ov *apis.ContainerVolumeMountDiskPostOverlay,
	ensure bool,
) (string, error) {
	return d.getPostOverlayDirWithPrefix(POST_OVERLAY_PREFIX_UPPER_DIR, pod, ctrId, vm, ov, ensure)
}

func (d diskPostOverlay) getPostOverlayMountpoint(pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, ov *apis.ContainerVolumeMountDiskPostOverlay) (string, error) {
	ctrMountHostPath, err := d.disk.GetRuntimeMountHostPath(pod, ctrId, vm)
	if err != nil {
		return "", errors.Wrap(err, "get runtime mount host_path")
	}
	// remove hostPath sub_directory path
	ctrMountHostPath = strings.TrimSuffix(ctrMountHostPath, vm.Disk.SubDirectory)
	mergedDir := filepath.Join(ctrMountHostPath, ov.ContainerTargetDir)
	if err := volume_mount.EnsureDir(mergedDir); err != nil {
		return "", errors.Wrap(err, "make merged mountpoint dir")
	}
	return mergedDir, nil
}

func (d diskPostOverlay) mountPostOverlay(pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, ov *apis.ContainerVolumeMountDiskPostOverlay) error {
	upperDir, err := d.getPostOverlayUpperDir(pod, ctrId, vm, ov, true)
	if err != nil {
		return errors.Wrapf(err, "get post overlay upper dir for container %s", ctrId)
	}

	workDir, err := d.getPostOverlayWorkDir(pod, ctrId, vm, ov, true)
	if err != nil {
		return errors.Wrapf(err, "get post overlay work dir for container %s", ctrId)
	}

	mergedDir, err := d.getPostOverlayMountpoint(pod, ctrId, vm, ov)
	if err != nil {
		return errors.Wrapf(err, "get post overlay mountpoint for container %s", ctrId)
	}

	return mountutils.MountOverlay(ov.HostLowerDir, upperDir, workDir, mergedDir)
}

func (d diskPostOverlay) unmountPostOverlay(pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, ov *apis.ContainerVolumeMountDiskPostOverlay, useLazy bool, cleanLayers bool) error {
	mergedDir, err := d.getPostOverlayMountpoint(pod, ctrId, vm, ov)
	if err != nil {
		return errors.Wrapf(err, "get post overlay mountpoint for container %s", ctrId)
	}
	if err := mountutils.Unmount(mergedDir, useLazy); err != nil {
		return errors.Wrapf(err, "unmount %s", mergedDir)
	}
	if cleanLayers {
		upperDir, err := d.getPostOverlayUpperDir(pod, ctrId, vm, ov, false)
		if err != nil {
			return errors.Wrapf(err, "get post overlay upper dir for container %s", ctrId)
		}
		if err := volume_mount.RemoveDir(upperDir); err != nil {
			return errors.Wrap(err, "remove upper dir")
		}

		workDir, err := d.getPostOverlayWorkDir(pod, ctrId, vm, ov, false)
		if err != nil {
			return errors.Wrapf(err, "get post overlay work dir for container %s", ctrId)
		}
		if err := volume_mount.RemoveDir(workDir); err != nil {
			return errors.Wrap(err, "remove work dir")
		}
	}
	return nil
}

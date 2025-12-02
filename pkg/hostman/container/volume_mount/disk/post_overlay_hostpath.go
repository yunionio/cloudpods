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

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/container/volume_mount"
	fileutils "yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/mountutils"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func init() {
	registerPostOverlayDriver(newPostOverlayHostPath())
}

func newPostOverlayHostPath() iDiskPostOverlayDriver {
	return &postOverlayHostPath{}
}

type postOverlayHostPath struct {
}

func (p postOverlayHostPath) GetType() apis.ContainerVolumeMountDiskPostOverlayType {
	return apis.CONTAINER_VOLUME_MOUNT_DISK_POST_OVERLAY_HOSTPATH
}

func (p postOverlayHostPath) getSingleFilePath(ov *apis.ContainerVolumeMountDiskPostOverlay) string {
	if len(ov.HostLowerDir) == 1 && fileutils.IsFile(ov.HostLowerDir[0]) {
		return ov.HostLowerDir[0]
	}
	return ""
}

func (p postOverlayHostPath) mountDir(d diskPostOverlay, pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, ov *apis.ContainerVolumeMountDiskPostOverlay) error {
	upperDir, err := d.getPostOverlayUpperDir(pod, ctrId, vm, ov, true)
	if err != nil {
		return errors.Wrapf(err, "get post overlay upper dir for container %s", ctrId)
	}

	workDir, err := d.getPostOverlayWorkDir(pod, ctrId, vm, ov, true)
	if err != nil {
		return errors.Wrapf(err, "get post overlay work dir for container %s", ctrId)
	}

	mergedDir, err := d.getPostOverlayMountpoint(pod, ctrId, vm, ov, true)
	if err != nil {
		return errors.Wrapf(err, "get post overlay mountpoint for container %s", ctrId)
	}

	if err := mountutils.MountOverlayWithFeatures(ov.HostLowerDir, upperDir, workDir, mergedDir, &mountutils.MountOverlayFeatures{
		MetaCopy: true,
	}); err != nil {
		return errors.Wrapf(err, "mount overlay dir for container %s", ctrId)
	}
	if err := volume_mount.ChangeDirOwnerDirectly(mergedDir, ov.FsUser, ov.FsGroup); err != nil {
		return errors.Wrapf(err, "change dir owner")
	}
	return nil
}

func (p postOverlayHostPath) getSingleFileMergedFilePath(mergedDir string, singleFilePath string) string {
	return filepath.Join(mergedDir, filepath.Base(singleFilePath))
}

func (p postOverlayHostPath) getSingleFileLowerDirFilePath(lowerDir string, singleFilePath string) string {
	return filepath.Join(lowerDir, filepath.Base(singleFilePath))
}

func (p postOverlayHostPath) mountSingleFile(singleFilePath string, d diskPostOverlay, pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, ov *apis.ContainerVolumeMountDiskPostOverlay) error {
	/*
		// 测试发现，如果 lowerDir 存在 bind mount 的文件，merge 后的目录中该文件为空，不是 bind mount 的 source 文件
			// 如果是单文件，创建一个目录，再把该文件 mount bind 过去到 lowerDir 里面
			lowerDir, err := d.getPostOverlayLowerDir(pod, ctrId, vm, ov, true)
			if err != nil {
				return errors.Wrapf(err, "get single file lower dir for container %s", ctrId)
			}
			lowerDirFilePath := p.getSingleFileLowerDirFilePath(lowerDir, singleFilePath)

			if !fileutils.Exists(lowerDirFilePath) {
				if err := volume_mount.TouchFile(lowerDirFilePath); err != nil {
					return errors.Wrapf(err, "touch file %s", lowerDirFilePath)
				}
			}
			if err := mountutils.MountBind(singleFilePath, lowerDirFilePath); err != nil {
				return errors.Wrapf(err, "mount bind %s to %s", singleFilePath, lowerDirFilePath)
			}
	*/

	// 单文件，直接把该文件的目录作为 lowerDir
	lowerDir := filepath.Dir(singleFilePath)

	// 把单文件的 lowerDir 挂载到 targetMergedDir，然后在把 singleFileMergedFilePath bind mount 到 mergedDir(mergedDir 其实是个文件路径)
	upperDir, err := d.getPostOverlayUpperDir(pod, ctrId, vm, ov, true)
	if err != nil {
		return errors.Wrapf(err, "get post overlay upper dir for container %s", ctrId)
	}
	workDir, err := d.getPostOverlayWorkDir(pod, ctrId, vm, ov, true)
	if err != nil {
		return errors.Wrapf(err, "get post overlay work dir for container %s", ctrId)
	}
	mergedDst, err := d.getPostOverlayMountpoint(pod, ctrId, vm, ov, false)
	if err != nil {
		return errors.Wrapf(err, "get post overlay mountpoint for container %s", ctrId)
	}
	targetMergedDir, err := d.getPostOverlayMergedDir(pod, ctrId, vm, ov, true)
	if err != nil {
		return errors.Wrapf(err, "get post overlay merged dir for container %s", ctrId)
	}
	if err := mountutils.MountOverlayWithFeatures([]string{lowerDir}, upperDir, workDir, targetMergedDir, &mountutils.MountOverlayFeatures{
		MetaCopy: true,
	}); err != nil {
		return errors.Wrapf(err, "mount overlay dir for container %s", ctrId)
	}
	singleFileMergedFilePath := p.getSingleFileMergedFilePath(targetMergedDir, singleFilePath)
	if err := volume_mount.ChangeDirOwnerDirectly(singleFileMergedFilePath, ov.FsUser, ov.FsGroup); err != nil {
		return errors.Wrapf(err, "change file %s owner", singleFilePath)
	}
	if !fileutils.Exists(mergedDst) {
		if err := volume_mount.TouchFile(mergedDst); err != nil {
			return errors.Wrapf(err, "touch file %s", mergedDst)
		}
	}
	if err := mountutils.MountBind(singleFileMergedFilePath, mergedDst); err != nil {
		return errors.Wrapf(err, "bind mount %s to %s", singleFileMergedFilePath, mergedDst)
	}
	return nil
}

func (p postOverlayHostPath) Mount(d diskPostOverlay, pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, ov *apis.ContainerVolumeMountDiskPostOverlay) error {
	// 支持单文件挂载
	singleFilePath := ""
	if len(ov.HostLowerDir) == 1 && fileutils.IsFile(ov.HostLowerDir[0]) {
		singleFilePath = ov.HostLowerDir[0]
	}
	if singleFilePath != "" {
		return p.mountSingleFile(singleFilePath, d, pod, ctrId, vm, ov)
	} else {
		return p.mountDir(d, pod, ctrId, vm, ov)
	}
}

func (p postOverlayHostPath) unmountDir(d diskPostOverlay, pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, ov *apis.ContainerVolumeMountDiskPostOverlay, useLazy bool, cleanLayers bool) error {
	mergedDir, err := d.getPostOverlayMountpoint(pod, ctrId, vm, ov, false)
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

func (p postOverlayHostPath) unmountSingleFile(singleFilePath string, d diskPostOverlay, pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, ov *apis.ContainerVolumeMountDiskPostOverlay, useLazy bool, cleanLayers bool) error {
	mergedFile, err := d.getPostOverlayMountpoint(pod, ctrId, vm, ov, false)
	if err != nil {
		return errors.Wrapf(err, "get post overlay mountpoint for container %s", ctrId)
	}
	targetMergedDir, err := d.getPostOverlayMergedDir(pod, ctrId, vm, ov, false)
	if err != nil {
		return errors.Wrapf(err, "get post overlay merged dir for container %s", ctrId)
	}
	singleFileMergedFilePath := p.getSingleFileMergedFilePath(targetMergedDir, singleFilePath)

	// 先 unbind mergedFile
	if err := mountutils.Unmount(mergedFile, false); err != nil {
		return errors.Wrapf(err, "unmount %s of single file %s", mergedFile, singleFileMergedFilePath)
	}
	// 如果 mergedFile 是空文件，则删除这个空文件，因为空文件很可能是挂载时创建的
	if procutils.IsEmptyFile(mergedFile) {
		if err := volume_mount.RemoveDir(mergedFile); err != nil {
			return errors.Wrapf(err, "remove empty file %s", mergedFile)
		}
	}

	if err := mountutils.Unmount(targetMergedDir, useLazy); err != nil {
		return errors.Wrapf(err, "unmount %s", targetMergedDir)
	}
	/*
		// 再 unbind lowerDir 里面的 singleFilePath
		lowerDir, err := d.getPostOverlayLowerDir(pod, ctrId, vm, ov, false)
		if err != nil {
			return errors.Wrapf(err, "get post overlay lower dir for container %s", ctrId)
		}
		lowerDirFilePath := p.getSingleFileLowerDirFilePath(lowerDir, singleFilePath)
		if err := mountutils.Unmount(lowerDirFilePath, false); err != nil {
			return errors.Wrapf(err, "unmount %s of single file %s", lowerDirFilePath, singleFileMergedFilePath)
		}
	*/
	if cleanLayers {
		upperDir, err := d.getPostOverlayUpperDir(pod, ctrId, vm, ov, false)
		if err != nil {
			return errors.Wrapf(err, "get post overlay upper dir for container %s", ctrId)
		}
		workDir, err := d.getPostOverlayWorkDir(pod, ctrId, vm, ov, false)
		if err != nil {
			return errors.Wrapf(err, "get post overlay work dir for container %s", ctrId)
		}

		for _, dir := range []string{upperDir, workDir, targetMergedDir} {
			if err := volume_mount.RemoveDir(dir); err != nil {
				return errors.Wrapf(err, "remove dir %s", dir)
			}
		}
	}
	return nil
}

func (p postOverlayHostPath) Unmount(d diskPostOverlay, pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, ov *apis.ContainerVolumeMountDiskPostOverlay, useLazy bool, cleanLayers bool) error {
	// 支持单文件卸载
	singleFilePath := ""
	if len(ov.HostLowerDir) == 1 && fileutils.IsFile(ov.HostLowerDir[0]) {
		singleFilePath = ov.HostLowerDir[0]
	}
	if singleFilePath != "" {
		return p.unmountSingleFile(singleFilePath, d, pod, ctrId, vm, ov, useLazy, cleanLayers)
	} else {
		return p.unmountDir(d, pod, ctrId, vm, ov, useLazy, cleanLayers)
	}
}

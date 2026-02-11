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
	"path/filepath"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/container/volume_mount"
)

func init() {
	registerPostOverlayDriver(newPostOverlayImage())
}

func newPostOverlayImage() iDiskPostOverlayDriver {
	return &postOverlayImage{}
}

type postOverlayImage struct {
}

func (i postOverlayImage) GetType() apis.ContainerVolumeMountDiskPostOverlayType {
	return apis.CONTAINER_VOLUME_MOUNT_DISK_POST_OVERLAY_IMAGE
}

func (i postOverlayImage) getImageInput(ov *apis.ContainerVolumeMountDiskPostOverlay) *apis.ContainerVolumeMountDiskPostImageOverlay {
	return ov.Image
}

func (i postOverlayImage) getCachedImagePaths(d diskPostOverlay, pod volume_mount.IPodInfo, img *apis.ContainerVolumeMountDiskPostImageOverlay, accquire bool) (map[string]string, map[string]*apis.HostLowerPath, error) {
	cachedImgDir, err := d.disk.getCachedImageDir(context.Background(), pod, img.Id, accquire)
	if err != nil {
		return nil, nil, errors.Wrap(err, "disk.getCachedImageDir")
	}
	hostLowerMap := make(map[string]*apis.HostLowerPath)
	if img.HostLowerMap != nil {
		hostLowerMap = img.HostLowerMap
	}
	result := make(map[string]string)
	lowerResult := make(map[string]*apis.HostLowerPath)
	for hostPathSuffix, ctrPath := range img.PathMap {
		hostPath := filepath.Join(cachedImgDir, hostPathSuffix)
		result[hostPath] = ctrPath
		hostLowerPath := hostLowerMap[hostPathSuffix]
		if hostLowerPath != nil {
			lowerResult[hostPath] = hostLowerPath
		}
	}
	return result, lowerResult, nil
}

// parseColonSeparatedPaths 解析冒号分隔的路径字符串，过滤空字符串并返回路径列表
func parseColonSeparatedPaths(pathStr string) []string {
	var paths []string
	for _, path := range strings.Split(pathStr, ":") {
		if path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}

func (i postOverlayImage) convertToDiskOV(ov *apis.ContainerVolumeMountDiskPostOverlay, hostPath, ctrPath string, hostLowerPath *apis.HostLowerPath, hostUpperDir string) *apis.ContainerVolumeMountDiskPostOverlay {
	hostLowerDir := []string{}
	if hostLowerPath != nil {
		// 添加 PrePath 中的路径
		hostLowerDir = append(hostLowerDir, parseColonSeparatedPaths(hostLowerPath.PrePath)...)
	}
	// 添加当前 hostPath
	hostLowerDir = append(hostLowerDir, hostPath)
	if hostLowerPath != nil {
		// 添加 PostPath 中的路径
		hostLowerDir = append(hostLowerDir, parseColonSeparatedPaths(hostLowerPath.PostPath)...)
	}
	return &apis.ContainerVolumeMountDiskPostOverlay{
		HostLowerDir:       hostLowerDir,
		HostUpperDir:       hostUpperDir,
		ContainerTargetDir: ctrPath,
		FsUser:             ov.FsUser,
		FsGroup:            ov.FsGroup,
		FlattenLayers:      ov.FlattenLayers,
	}
}

func (i postOverlayImage) withAction(
	vm *hostapi.ContainerVolumeMount,
	d diskPostOverlay, pod volume_mount.IPodInfo, ov *apis.ContainerVolumeMountDiskPostOverlay,
	af func(iDiskPostOverlayDriver, *apis.ContainerVolumeMountDiskPostOverlay) error,
	accquire bool) error {
	img := i.getImageInput(ov)
	paths, hostLowerPaths, err := i.getCachedImagePaths(d, pod, img, accquire)
	if err != nil {
		return errors.Wrapf(err, "get cached image paths")
	}
	hostUpperDir := ""
	if img.UpperConfig != nil {
		config := img.UpperConfig
		if config.Disk.SubPath == "" {
			return errors.Errorf("sub_path of upper config is empty")
		}
		hostPath, err := d.disk.GetHostDiskRootPath(pod, vm)
		if err != nil {
			return errors.Wrap(err, "get host disk root path")
		}
		hostUpperDir = filepath.Join(hostPath, vm.Disk.SubDirectory, config.Disk.SubPath)
	}
	for hostPath, ctrPath := range paths {
		hostLowerPath := hostLowerPaths[hostPath]
		dov := i.convertToDiskOV(ov, hostPath, ctrPath, hostLowerPath, hostUpperDir)
		drv := d.getDriver(dov)
		if err := af(drv, dov); err != nil {
			return errors.Wrapf(err, "host path %s to %s", hostPath, ctrPath)
		}
	}
	return nil
}

func (i postOverlayImage) Mount(d diskPostOverlay, pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, ov *apis.ContainerVolumeMountDiskPostOverlay) error {
	return i.withAction(vm, d, pod, ov,
		func(driver iDiskPostOverlayDriver, dov *apis.ContainerVolumeMountDiskPostOverlay) error {
			return driver.Mount(d, pod, ctrId, vm, dov)
		}, true)
}

func (i postOverlayImage) Unmount(d diskPostOverlay, pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, ov *apis.ContainerVolumeMountDiskPostOverlay, useLazy bool, clearLayers bool) error {
	return i.withAction(vm, d, pod, ov,
		func(driver iDiskPostOverlayDriver, dov *apis.ContainerVolumeMountDiskPostOverlay) error {
			return driver.Unmount(d, pod, ctrId, vm, dov, useLazy, clearLayers)
		}, false)
}

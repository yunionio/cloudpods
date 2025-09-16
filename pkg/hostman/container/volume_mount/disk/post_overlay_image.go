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

func (i postOverlayImage) getCachedImagePaths(d diskPostOverlay, pod volume_mount.IPodInfo, img *apis.ContainerVolumeMountDiskPostImageOverlay) (map[string]string, error) {
	cachedImgDir, err := d.disk.getCachedImageDir(context.Background(), pod, img.Id)
	if err != nil {
		return nil, errors.Wrap(err, "disk.getCachedImageDir")
	}
	result := make(map[string]string)
	for hostPathSuffix, ctrPath := range img.PathMap {
		hostPath := filepath.Join(cachedImgDir, hostPathSuffix)
		result[hostPath] = ctrPath
	}
	return result, nil
}

func (i postOverlayImage) convertToDiskOV(ov *apis.ContainerVolumeMountDiskPostOverlay, hostPath, ctrPath string) *apis.ContainerVolumeMountDiskPostOverlay {
	return &apis.ContainerVolumeMountDiskPostOverlay{
		HostLowerDir:       []string{hostPath},
		ContainerTargetDir: ctrPath,
	}
}

func (i postOverlayImage) withAction(
	d diskPostOverlay, pod volume_mount.IPodInfo, ov *apis.ContainerVolumeMountDiskPostOverlay,
	af func(iDiskPostOverlayDriver, *apis.ContainerVolumeMountDiskPostOverlay) error) error {
	img := i.getImageInput(ov)
	paths, err := i.getCachedImagePaths(d, pod, img)
	if err != nil {
		return errors.Wrapf(err, "get cached image paths")
	}
	for hostPath, ctrPath := range paths {
		dov := i.convertToDiskOV(ov, hostPath, ctrPath)
		drv := d.getDriver(dov)
		if err := af(drv, dov); err != nil {
			return errors.Wrapf(err, "host path %s to %s", hostPath, ctrPath)
		}
	}
	return nil
}

func (i postOverlayImage) Mount(d diskPostOverlay, pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, ov *apis.ContainerVolumeMountDiskPostOverlay) error {
	return i.withAction(d, pod, ov,
		func(driver iDiskPostOverlayDriver, dov *apis.ContainerVolumeMountDiskPostOverlay) error {
			return driver.Mount(d, pod, ctrId, vm, dov)
		})
}

func (i postOverlayImage) Unmount(d diskPostOverlay, pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, ov *apis.ContainerVolumeMountDiskPostOverlay, useLazy bool, clearLayers bool) error {
	return i.withAction(d, pod, ov,
		func(driver iDiskPostOverlayDriver, dov *apis.ContainerVolumeMountDiskPostOverlay) error {
			return driver.Unmount(d, pod, ctrId, vm, dov, useLazy, clearLayers)
		})
}

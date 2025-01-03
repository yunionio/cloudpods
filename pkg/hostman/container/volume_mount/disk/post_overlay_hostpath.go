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
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/container/volume_mount"
	"yunion.io/x/onecloud/pkg/util/mountutils"
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

func (p postOverlayHostPath) Mount(d diskPostOverlay, pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, ov *apis.ContainerVolumeMountDiskPostOverlay) error {
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

	return mountutils.MountOverlayWithFeatures(ov.HostLowerDir, upperDir, workDir, mergedDir, &mountutils.MountOverlayFeatures{
		MetaCopy: true,
	})
}

func (p postOverlayHostPath) Unmount(d diskPostOverlay, pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, ov *apis.ContainerVolumeMountDiskPostOverlay, useLazy bool, cleanLayers bool) error {
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

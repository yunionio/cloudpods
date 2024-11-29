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

	"yunion.io/x/pkg/errors"

	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	container_storage "yunion.io/x/onecloud/pkg/hostman/container/storage"
	"yunion.io/x/onecloud/pkg/hostman/container/volume_mount"
	"yunion.io/x/onecloud/pkg/util/mountutils"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func (d disk) getOverlayDir(pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, oType string, upperDir string) string {
	baseDir := strings.TrimPrefix(upperDir, pod.GetVolumesDir())
	return filepath.Join(pod.GetVolumesOverlayDir(), vm.Disk.Id, ctrId, oType, baseDir)
}

func (d disk) getOverlayWorkDir(upperDir string) string {
	return fmt.Sprintf("%s-work", upperDir)
}

func (d disk) getOverlayMergedDir(pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount, upperDir string) string {
	return d.getOverlayDir(pod, ctrId, vm, "_merged_", upperDir)
}

type iDiskOverlay interface {
	mount(d disk, pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error
	unmount(d disk, pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error
}

type diskOverlayDir struct{}

func newDiskOverlayDir() iDiskOverlay {
	return &diskOverlayDir{}
}

func (dod diskOverlayDir) mount(d disk, pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error {
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

	return mountutils.MountOverlay(lowerDir, upperDir, workDir, mergedDir)
}

func (dod diskOverlayDir) unmount(d disk, pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error {
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

func (di diskOverlayImage) mount(d disk, pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error {
	if err := d.doTemplateOverlayAction(context.Background(), pod, ctrId, vm, newDiskOverlayDir().mount); err != nil {
		return errors.Wrapf(err, "mount template overlay")
	}
	return nil
}

func (di diskOverlayImage) unmount(d disk, pod volume_mount.IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error {
	if err := d.doTemplateOverlayAction(context.Background(), pod, ctrId, vm, newDiskOverlayDir().unmount); err != nil {
		return errors.Wrapf(err, "unmount template overlay")
	}
	return nil
}

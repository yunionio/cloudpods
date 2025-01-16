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
	"fmt"
	"path/filepath"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	container_storage "yunion.io/x/onecloud/pkg/hostman/container/storage"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func init() {
	RegisterDriver(newCephFS())
}

type cephFS struct{}

func (h cephFS) Mount(pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error {
	dir, err := h.GetRuntimeMountHostPath(pod, ctrId, vm)
	if err != nil {
		return err
	}
	if err := EnsureDir(dir); err != nil {
		return errors.Wrap(err, "EnsureDir")
	}
	if err := procutils.NewRemoteCommandAsFarAsPossible("mountpoint", dir).Run(); err == nil {
		log.Warningf("mountpoint %s is already mounted", dir)
		return nil
	}
	options := fmt.Sprintf("name=%s,secret=%s", vm.CephFS.Name, vm.CephFS.Secret)
	if vm.ReadOnly {
		options += ",ro"
	}
	args := []string{
		"-t", "ceph",
		fmt.Sprintf("%s:%s", vm.CephFS.MonHost, vm.CephFS.Path), dir,
		"-o", options,
	}
	out, err := procutils.NewRemoteCommandAsFarAsPossible("mount", args...).Output()
	if err != nil {
		return errors.Wrapf(err, "mount %s: %s", dir, out)
	}
	return nil
}

func (h cephFS) Unmount(pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error {
	dir, err := h.GetRuntimeMountHostPath(pod, ctrId, vm)
	if err != nil {
		return errors.Wrap(err, "GetRuntimeMountHostPath")
	}
	if err := container_storage.Unmount(dir); err != nil {
		return errors.Wrapf(err, "unmount %s", dir)
	}
	if out, err := procutils.NewRemoteCommandAsFarAsPossible("rm", "-fd", dir).Output(); err != nil {
		return errors.Wrapf(err, "rm -fd %s: %s", dir, string(out))
	}
	return nil
}

func newCephFS() IVolumeMount {
	return &cephFS{}
}

func (h cephFS) GetType() apis.ContainerVolumeMountType {
	return apis.CONTAINER_VOLUME_MOUNT_TYPE_CEPHF_FS
}

func (d cephFS) InjectUsageTags(usage *ContainerVolumeMountUsage, vol *hostapi.ContainerVolumeMount) {
	if vol.CephFS != nil {
		usage.Tags["cephfs_id"] = vol.CephFS.Id
	}
}

func (h cephFS) GetRuntimeMountHostPath(pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) (string, error) {
	if vm.CephFS == nil {
		return "", fmt.Errorf("cephfs is nil")
	}
	return filepath.Join(pod.GetVolumesDir(), vm.CephFS.Id), nil
}

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
	"os"
	"path/filepath"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func init() {
	RegisterDriver(newHostLocal())
}

type hostLocal struct{}

func (h hostLocal) Mount(pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error {
	return nil
}

func (h hostLocal) Unmount(pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error {
	return nil
}

func newHostLocal() IVolumeMount {
	return &hostLocal{}
}

func (h hostLocal) GetType() apis.ContainerVolumeMountType {
	return apis.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH
}

func (h hostLocal) GetRuntimeMountHostPath(pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) (string, error) {
	host := vm.HostPath
	if host == nil {
		return "", httperrors.NewNotEmptyError("host_local is nil")
	}
	if vm.FsUser != nil || vm.FsGroup != nil {
		return "", httperrors.NewInputParameterError("cannot use fs_user and fs_group for host_local volume")
	}
	clean, err := fileutils2.CleanHostBindPath(host.Path)
	if err != nil {
		return "", httperrors.NewInputParameterError("%v", err)
	}
	host.Path = clean
	switch host.Type {
	case "", apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_FILE:
		return h.getFilePath(host)
	case apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY:
		return h.getDirectoryPath(host)
	}
	return "", httperrors.NewInputParameterError("unsupported type %q", host.Type)
}

func (h hostLocal) getFilePath(input *apis.ContainerVolumeMountHostPath) (string, error) {
	if input.Type != "" && input.Type != apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_FILE {
		return "", httperrors.NewInputParameterError("unsupported type %q", input.Type)
	}
	if err := h.ensureHostPath(input.Path, false, input); err != nil {
		return "", err
	}
	return input.Path, nil
}

func (h hostLocal) getDirectoryPath(input *apis.ContainerVolumeMountHostPath) (string, error) {
	if input.Type != apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY {
		return "", httperrors.NewInputParameterError("unsupported type %q", input.Type)
	}
	if err := h.ensureHostPath(input.Path, true, input); err != nil {
		return "", err
	}
	return input.Path, nil
}

func (h hostLocal) ensureHostPath(path string, isDir bool, input *apis.ContainerVolumeMountHostPath) error {
	fi, err := procutils.RemoteStat(path)
	if err != nil {
		if errors.Cause(err) != os.ErrNotExist {
			return errors.Wrapf(err, "stat %s", path)
		}
		if !input.AutoCreate {
			return errors.Wrapf(err, "path %s does not exist and auto_create is not set", path)
		}
		if isDir {
			if err := EnsureDir(path); err != nil {
				return err
			}
		} else {
			if err := EnsureDir(filepath.Dir(path)); err != nil {
				return errors.Wrapf(err, "create parent directory for %s", path)
			}
			if err := TouchFile(path); err != nil {
				return err
			}
		}
		return applyHostPathAutoCreateConfig(path, input.AutoCreateConfig)
	}
	if isDir && !fi.IsDir() {
		return httperrors.NewInputParameterError("%s is not a directory", path)
	}
	if !isDir && fi.IsDir() {
		return httperrors.NewInputParameterError("%s is a directory", path)
	}
	return nil
}

func applyHostPathAutoCreateConfig(path string, cfg *apis.ContainerVolumeMountHostPathAutoCreateConfig) error {
	if cfg == nil {
		return nil
	}
	if cfg.Permissions != "" {
		_, canon, err := fileutils2.ParseUnixFileMode(cfg.Permissions)
		if err != nil {
			return errors.Wrap(err, "permissions")
		}
		if err := Chmod(path, canon); err != nil {
			return err
		}
	}
	if cfg.Uid > 0 || cfg.Gid > 0 {
		var uid, gid *int64
		if cfg.Uid > 0 {
			u := int64(cfg.Uid)
			uid = &u
		}
		if cfg.Gid > 0 {
			g := int64(cfg.Gid)
			gid = &g
		}
		if err := ChangeDirOwnerDirectly(path, uid, gid); err != nil {
			return err
		}
	}
	return nil
}

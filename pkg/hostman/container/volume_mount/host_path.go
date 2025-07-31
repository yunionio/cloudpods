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

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/httperrors"
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
	switch host.Type {
	case "", apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_FILE:
		return h.getFilePath(host)
	case apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY:
		return h.getDirectoryPath(host)
	}
	return "", httperrors.NewInputParameterError("unsupported type %q", host.Type)
}

func (h hostLocal) getFilePath(input *apis.ContainerVolumeMountHostPath) (string, error) {
	if input.Type != apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_FILE {
		return "", httperrors.NewInputParameterError("unsupported type %q", input.Type)
	}
	filePath := input.Path

	// 检查文件是否存在
	checkCmd := fmt.Sprintf("test -f '%s'", filePath)
	if _, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", checkCmd).Output(); err != nil {
		// 文件不存在，需要创建
		if !input.AutoCreate {
			return "", errors.Wrapf(err, "file %s does not exist and no auto_create specified", filePath)
		}

		// 先确保父目录存在
		parentDirCmd := fmt.Sprintf("mkdir -p '%s'", filepath.Dir(filePath))
		if out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", parentDirCmd).Output(); err != nil {
			return "", errors.Wrapf(err, "create parent directory for %s: %s", filePath, out)
		}

		// 创建文件
		createCmd := fmt.Sprintf("touch '%s'", filePath)
		if out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", createCmd).Output(); err != nil {
			return "", errors.Wrapf(err, "create file %s: %s", filePath, out)
		}

		if input.AutoCreateConfig != nil {
			// 设置权限
			if input.AutoCreateConfig.Permissions != "" {
				chmodCmd := fmt.Sprintf("chmod %s '%s'", input.AutoCreateConfig.Permissions, filePath)
				if out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", chmodCmd).Output(); err != nil {
					return "", errors.Wrapf(err, "chmod %s %s: %s", input.AutoCreateConfig.Permissions, filePath, out)
				}
			}

			// 设置所有者
			if input.AutoCreateConfig.Uid > 0 || input.AutoCreateConfig.Gid > 0 {
				chownCmd := fmt.Sprintf("chown %d:%d '%s'", input.AutoCreateConfig.Uid, input.AutoCreateConfig.Gid, filePath)
				if out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", chownCmd).Output(); err != nil {
					return "", errors.Wrapf(err, "chown %d:%d %s: %s", input.AutoCreateConfig.Uid, input.AutoCreateConfig.Gid, filePath, out)
				}
			}
		}
	}

	return filePath, nil
}

func (h hostLocal) getDirectoryPath(input *apis.ContainerVolumeMountHostPath) (string, error) {
	if input.Type != apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY {
		return "", httperrors.NewInputParameterError("unsupported type %q", input.Type)
	}
	dirPath := input.Path

	// 检查目录是否存在
	checkCmd := fmt.Sprintf("test -d '%s'", dirPath)
	if _, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", checkCmd).Output(); err != nil {
		if !input.AutoCreate {
			return "", errors.Wrapf(err, "dir %s does not exist and no auto_create specified", dirPath)
		}
		// 创建目录
		createCmd := fmt.Sprintf("mkdir -p '%s'", dirPath)
		if out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", createCmd).Output(); err != nil {
			return "", errors.Wrapf(err, "create directory %s: %s", dirPath, out)
		}

		if input.AutoCreateConfig != nil {
			// 设置权限
			if input.AutoCreateConfig.Permissions != "" {
				chmodCmd := fmt.Sprintf("chmod %s '%s'", input.AutoCreateConfig.Permissions, dirPath)
				if out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", chmodCmd).Output(); err != nil {
					return "", errors.Wrapf(err, "chmod %s %s: %s", input.AutoCreateConfig.Permissions, dirPath, out)
				}
			}

			// 设置所有者
			if input.AutoCreateConfig.Uid > 0 || input.AutoCreateConfig.Gid > 0 {
				chownCmd := fmt.Sprintf("chown %d:%d '%s'", input.AutoCreateConfig.Uid, input.AutoCreateConfig.Gid, dirPath)
				if out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", chownCmd).Output(); err != nil {
					return "", errors.Wrapf(err, "chown %d:%d %s: %s", input.AutoCreateConfig.Uid, input.AutoCreateConfig.Gid, dirPath, out)
				}
			}
		}
	}

	return dirPath, nil
}

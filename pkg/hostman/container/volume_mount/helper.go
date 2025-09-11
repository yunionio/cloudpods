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

	"yunion.io/x/pkg/errors"

	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func TouchFile(file string) error {
	out, err := procutils.NewRemoteCommandAsFarAsPossible("touch", file).Output()
	if err != nil {
		return errors.Wrapf(err, "touch %s: %s", file, out)
	}
	return nil
}

func EnsureDir(dir string) error {
	out, err := procutils.NewRemoteCommandAsFarAsPossible("mkdir", "-p", dir).Output()
	if err != nil {
		return errors.Wrapf(err, "mkdir -p %s: %s", dir, out)
	}
	return nil
}

func RemoveDir(dir string) error {
	out, err := procutils.NewRemoteCommandAsFarAsPossible("rm", "-rf", dir).Output()
	if err != nil {
		return errors.Wrapf(err, "rm -rf %s: %s", dir, out)
	}
	return nil
}

func ChangeDirOwner(pod IPodInfo, drv IVolumeMount, ctrId string, vol *hostapi.ContainerVolumeMount) error {
	if vol.FsUser == nil && vol.FsGroup == nil {
		return errors.Errorf("specify fs_user or fs_group")
	}
	hostPath, err := drv.GetRuntimeMountHostPath(pod, ctrId, vol)
	if err != nil {
		return errors.Wrap(err, "GetRuntimeMountHostPath")
	}
	return ChangeDirOwnerDirectly(hostPath, vol.FsUser, vol.FsGroup)
}

func ChangeDirOwnerDirectly(hostPath string, fsUser, fsGroup *int64) error {
	args := ""
	if fsUser != nil {
		args = fmt.Sprintf("%d", *fsUser)
	}
	if fsGroup != nil {
		args = fmt.Sprintf("%s:%d", args, *fsGroup)
	}
	if args == "" {
		return nil
	}
	out, err := procutils.NewRemoteCommandAsFarAsPossible("chown", args, hostPath).Output()
	if err != nil {
		return errors.Wrapf(err, "chown %s %s: %s", args, hostPath, string(out))
	}
	return nil
}

func CopyFile(src, dst string) error {
	out, err := procutils.NewRemoteCommandAsFarAsPossible("cp", src, dst).Output()
	if err != nil {
		return errors.Wrapf(err, "cp %s %s: %s", src, dst, string(out))
	}
	return nil
}

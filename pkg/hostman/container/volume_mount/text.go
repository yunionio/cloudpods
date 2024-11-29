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
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func init() {
	RegisterDriver(newText())
}

type text struct {
}

func newText() IVolumeMount {
	return &text{}
}

func (t text) GetType() apis.ContainerVolumeMountType {
	return apis.CONTAINER_VOLUME_MOUNT_TYPE_TEXT
}

func (t text) GetRuntimeMountHostPath(pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) (string, error) {
	ti := vm.Text
	if ti == nil {
		return "", httperrors.NewNotEmptyError("text is nil")
	}
	if err := EnsureDir(pod.GetVolumesDir()); err != nil {
		return "", errors.Wrapf(err, "mkdir %s", pod.GetVolumesDir())
	}
	mntPath := filepath.Join(pod.GetVolumesDir(), fmt.Sprintf("%s-%s", ctrId, strings.ReplaceAll(vm.MountPath, "/", "_")))
	if err := t.writeContent(ti, mntPath); err != nil {
		return "", errors.Wrapf(err, "write content %s to %s", ti, mntPath)
	}
	return mntPath, nil
}

func (t text) writeContent(ti *apis.ContainerVolumeMountText, path string) error {
	b64Content := base64.StdEncoding.EncodeToString([]byte(ti.Content))
	out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", fmt.Sprintf("echo %s | base64 -d > %s", b64Content, path)).Output()
	if err != nil {
		return errors.Wrapf(err, "write content to %s: %s", path, out)
	}
	return nil
}

func (t text) Mount(pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error {
	return nil
}

func (t text) Unmount(pod IPodInfo, ctrId string, vm *hostapi.ContainerVolumeMount) error {
	return nil
}

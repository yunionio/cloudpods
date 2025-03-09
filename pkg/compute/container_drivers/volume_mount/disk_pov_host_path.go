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
	"context"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type povHostPath struct {
}

func newDiskPostOverlayHostPath() iDiskPostOverlay {
	return &povHostPath{}
}

func (p povHostPath) validateData(ctx context.Context, userCred mcclient.TokenCredential, ov *apis.ContainerVolumeMountDiskPostOverlay) error {
	if len(ov.HostLowerDir) == 0 {
		return httperrors.NewNotEmptyError("host_lower_dir is required")
	}
	for i, hld := range ov.HostLowerDir {
		if len(hld) == 0 {
			return httperrors.NewNotEmptyError("host_lower_dir %d is empty", i)
		}
	}
	if len(ov.ContainerTargetDir) == 0 {
		return httperrors.NewNotEmptyError("container_target_dir is required")
	}
	return nil
}

func (p povHostPath) getContainerTargetDirs(ov *apis.ContainerVolumeMountDiskPostOverlay) []string {
	return []string{ov.ContainerTargetDir}
}

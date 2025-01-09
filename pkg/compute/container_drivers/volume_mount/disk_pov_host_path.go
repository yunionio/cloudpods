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

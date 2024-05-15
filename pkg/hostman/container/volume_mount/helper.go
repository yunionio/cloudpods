package volume_mount

import (
	"fmt"

	"yunion.io/x/pkg/errors"

	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func ChangeDirOwner(pod IPodInfo, drv IVolumeMount, ctrId string, vol *hostapi.ContainerVolumeMount) error {
	if vol.FsUser == nil && vol.FsGroup == nil {
		return errors.Errorf("specify fs_user or fs_group")
	}
	hostPath, err := drv.GetRuntimeMountHostPath(pod, ctrId, vol)
	if err != nil {
		return errors.Wrap(err, "GetRuntimeMountHostPath")
	}
	args := ""
	if vol.FsUser != nil {
		args = fmt.Sprintf("%d", *vol.FsUser)
	}
	if vol.FsGroup != nil {
		args = fmt.Sprintf("%s:%d", args, *vol.FsGroup)
	}
	out, err := procutils.NewRemoteCommandAsFarAsPossible("chown", "-R", args, hostPath).Output()
	if err != nil {
		return errors.Wrapf(err, "chown -R %s %s: %s", args, hostPath, string(out))
	}
	return nil
}

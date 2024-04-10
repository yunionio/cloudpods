package volume_mount

import (
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
	switch host.Type {
	case "", apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_FILE:
		return host.Path, nil
	case apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY:
		return h.getDirectoryPath(host)
	}
	return "", httperrors.NewInputParameterError("unsupported type %q", host.Type)
}

func (h hostLocal) getDirectoryPath(input *apis.ContainerVolumeMountHostPath) (string, error) {
	if input.Type != apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY {
		return "", httperrors.NewInputParameterError("unsupported type %q", input.Type)
	}
	dirPath := input.Path
	out, err := procutils.NewRemoteCommandAsFarAsPossible("mkdir", "-p", dirPath).Output()
	if err != nil {
		return "", errors.Wrapf(err, "mkdir -p %s: %s", dirPath, out)
	}
	return dirPath, nil
}

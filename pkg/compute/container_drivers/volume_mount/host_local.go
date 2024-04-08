package volume_mount

import (
	"context"

	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	models.RegisterContainerVolumeMountDriver(newHostLocal())
}

type hostLocal struct{}

func newHostLocal() models.IContainerVolumeMountDriver {
	return &hostLocal{}
}

func (h hostLocal) GetType() apis.ContainerVolumeMountType {
	return apis.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH
}

func (h hostLocal) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, pod *models.SGuest, vm *apis.ContainerVolumeMount) (*apis.ContainerVolumeMount, error) {
	return vm, h.ValidatePodCreateData(ctx, userCred, vm, nil)
}

func (h hostLocal) ValidatePodCreateData(ctx context.Context, userCred mcclient.TokenCredential, vm *apis.ContainerVolumeMount, input *api.ServerCreateInput) error {
	hp := vm.HostPath
	if hp == nil {
		return httperrors.NewNotEmptyError("host_path is nil")
	}
	if hp.Type == "" {
		hp.Type = apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_FILE
	}
	if !sets.NewString(
		string(apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_FILE),
		string(apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY)).Has(string(hp.Type)) {
		return httperrors.NewInputParameterError("unsupported type %s", hp.Type)
	}
	if hp.Path == "" {
		return httperrors.NewNotEmptyError("path is required")
	}
	return nil
}

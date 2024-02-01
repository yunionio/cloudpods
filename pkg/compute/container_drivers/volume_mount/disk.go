package volume_mount

import (
	"context"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	models.RegisterContainerVolumeMountDriver(newDisk())
}

type disk struct{}

func newDisk() models.IContainerVolumeMountDriver {
	return &disk{}
}

func (d disk) GetType() apis.ContainerVolumeMountType {
	return apis.CONTAINER_VOLUME_MOUNT_TYPE_DISK
}

func (d disk) validateCreateData(ctx context.Context, userCred mcclient.TokenCredential, vm *apis.ContainerVolumeMount) (*apis.ContainerVolumeMount, error) {
	disk := vm.Disk
	if disk == nil {
		return nil, httperrors.NewNotEmptyError("disk is nil")
	}
	if disk.Index == nil && disk.Id == "" {
		return nil, httperrors.NewNotEmptyError("one of index or id is required")
	}
	if disk.Index != nil {
		if *disk.Index < 0 {
			return nil, httperrors.NewInputParameterError("index is less than 0")
		}
	}
	return vm, nil
}

func (d disk) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, pod *models.SGuest, vm *apis.ContainerVolumeMount) (*apis.ContainerVolumeMount, error) {
	if _, err := d.validateCreateData(ctx, userCred, vm); err != nil {
		return nil, err
	}
	disks, err := pod.GetDisks()
	if err != nil {
		return nil, errors.Wrap(err, "get pod disks")
	}
	disk := vm.Disk
	if disk.Index != nil {
		diskIndex := *disk.Index
		if diskIndex >= len(disks) {
			return nil, httperrors.NewInputParameterError("disk.index %d is large than disk size %d", diskIndex, len(disks))
		}
		vm.Disk.Id = disks[diskIndex].GetId()
		// remove index
		vm.Disk.Index = nil
	} else {
		if disk.Id == "" {
			return nil, httperrors.NewNotEmptyError("disk.id is empty")
		}
		foundDisk := false
		for _, d := range disks {
			if d.GetId() == disk.Id || d.GetName() == disk.Id {
				disk.Id = d.GetId()
				foundDisk = true
				break
			}
		}
		if !foundDisk {
			return nil, httperrors.NewNotFoundError("not found pod disk by %s", disk.Id)
		}
	}
	return vm, nil
}

func (d disk) ValidatePodCreateData(ctx context.Context, userCred mcclient.TokenCredential, vm *apis.ContainerVolumeMount, input *api.ServerCreateInput) error {
	if _, err := d.validateCreateData(ctx, userCred, vm); err != nil {
		return err
	}
	disk := vm.Disk
	if disk.Id != "" {
		return httperrors.NewInputParameterError("can't specify disk_id %s when creating pod", disk.Id)
	}
	if disk.Index == nil {
		return httperrors.NewNotEmptyError("disk.index is required")
	}
	diskIndex := *disk.Index
	disks := input.Disks
	if diskIndex < 0 {
		return httperrors.NewInputParameterError("disk.index %d is less than 0", diskIndex)
	}
	if diskIndex >= len(disks) {
		return httperrors.NewInputParameterError("disk.index %d is large than disk size %d", diskIndex, len(disks))
	}
	return nil
}

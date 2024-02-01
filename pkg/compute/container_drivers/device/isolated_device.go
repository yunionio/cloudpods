package device

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	models.RegisterContainerDeviceDriver(newIsolatedDevice())
}

type isolatedDevice struct{}

func newIsolatedDevice() models.IContainerDeviceDriver {
	return &isolatedDevice{}
}

func (i isolatedDevice) GetType() apis.ContainerDeviceType {
	return apis.CONTAINER_DEVICE_TYPE_ISOLATED_DEVICE
}

func (i isolatedDevice) validateCreateData(dev *api.ContainerDevice) error {
	isoDev := dev.IsolatedDevice
	if isoDev == nil {
		return httperrors.NewNotEmptyError("isolated_device is nil")
	}
	if isoDev.Index == nil && isoDev.Id == "" {
		return httperrors.NewNotEmptyError("one of index or id is required")
	}
	if isoDev.Index != nil {
		if *isoDev.Index < 0 {
			return httperrors.NewInputParameterError("index is less than 0")
		}
	}
	return nil
}

func (i isolatedDevice) ValidatePodCreateData(ctx context.Context, userCred mcclient.TokenCredential, dev *api.ContainerDevice, input *api.ServerCreateInput) error {
	if err := i.validateCreateData(dev); err != nil {
		return errors.Wrapf(err, "validate create data %s", jsonutils.Marshal(dev))
	}
	isoDev := dev.IsolatedDevice
	if isoDev.Id != "" {
		return httperrors.NewInputParameterError("can't specify id %s when creating pod", isoDev.Id)
	}
	if isoDev.Index == nil {
		return httperrors.NewNotEmptyError("index is required")
	}
	inputDevs := input.IsolatedDevices
	if *isoDev.Index >= len(inputDevs) {
		return httperrors.NewInputParameterError("disk.index %d is large than disk size %d", isoDev.Index, len(inputDevs))
	}
	return nil
}

func (i isolatedDevice) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, pod *models.SGuest, dev *api.ContainerDevice) (*api.ContainerDevice, error) {
	if err := i.validateCreateData(dev); err != nil {
		return nil, errors.Wrapf(err, "validate create data %s", jsonutils.Marshal(dev))
	}
	isoDev := dev.IsolatedDevice
	podDevs, err := pod.GetIsolatedDevices()
	if err != nil {
		return nil, errors.Wrap(err, "get isolated devices")
	}
	if isoDev.Index != nil {
		index := *isoDev.Index
		if index >= len(podDevs) {
			return nil, httperrors.NewInputParameterError("index %d is large than isolated device size %d", index, len(podDevs))
		}
		isoDev.Id = podDevs[index].GetId()
		// remove index
		isoDev.Index = nil
	} else {
		if isoDev.Id == "" {
			return nil, httperrors.NewNotEmptyError("id is empty")
		}
		foundDisk := false
		for _, d := range podDevs {
			if d.GetId() == isoDev.Id || d.GetName() == isoDev.Id {
				isoDev.Id = d.GetId()
				foundDisk = true
				devType := d.DevType
				if !sets.NewString(api.VALID_CONTAINER_DEVICE_TYPES...).Has(devType) {
					return nil, httperrors.NewInputParameterError("device type %s is not supported by container", devType)
				}
				break
			}
		}
		if !foundDisk {
			return nil, httperrors.NewNotFoundError("not found pod device by %s", isoDev.Id)
		}
	}
	dev.IsolatedDevice = isoDev
	return dev, nil
}

func (i isolatedDevice) ToHostDevice(dev *api.ContainerDevice) (*hostapi.ContainerDevice, error) {
	input := dev.IsolatedDevice
	isoDevObj, err := models.IsolatedDeviceManager.FetchById(input.Id)
	if err != nil {
		return nil, errors.Wrapf(err, "Fetch isolated device by id %s", input.Id)
	}
	isoDev := isoDevObj.(*models.SIsolatedDevice)
	return &hostapi.ContainerDevice{
		Type: dev.Type,
		IsolatedDevice: &hostapi.ContainerIsolatedDevice{
			Id:         isoDev.GetId(),
			Addr:       isoDev.Addr,
			Path:       isoDev.DevicePath,
			DeviceType: isoDev.DevType,
		},
	}, nil
}

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

package device

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

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
	if isoDev.CDI != nil {
		if isoDev.CDI.Kind == "" {
			return httperrors.NewNotEmptyError("cdk.kind is empty")
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
	podDevs, err := pod.GetGuestIsolatedDevices()
	if err != nil {
		return nil, errors.Wrap(err, "get isolated devices")
	}
	if isoDev.Index != nil {
		index := *isoDev.Index
		if index >= len(podDevs) {
			return nil, httperrors.NewInputParameterError("index %d is large than isolated device size %d", index, len(podDevs))
		}
		isoDev.Id = podDevs[index].IsolatedDeviceId
		// remove index
		isoDev.Index = nil
	} else {
		if isoDev.Id == "" {
			return nil, httperrors.NewNotEmptyError("id is empty")
		}
		foundDisk := false
		for i := range podDevs {
			d := podDevs[i].GetIsolatedDevice()
			if d.GetId() == isoDev.Id || d.GetName() == isoDev.Id {
				isoDev.Id = d.GetId()
				foundDisk = true
				host := d.GetHost()
				if host.HostType != api.HOST_TYPE_CONTAINER {
					return nil, httperrors.NewInputParameterError("device %s is not supported by container", isoDev.Id)
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

func (i isolatedDevice) ToHostDevice(dev *api.ContainerDevice, guestId string) (*hostapi.ContainerDevice, error) {
	input := dev.IsolatedDevice
	isoDevObj, err := models.IsolatedDeviceManager.FetchById(input.Id)
	if err != nil {
		return nil, errors.Wrapf(err, "Fetch isolated device by id %s", input.Id)
	}
	isoDev := isoDevObj.(*models.SIsolatedDevice)
	gdev, err := isoDev.GetGuestIsolatedDevice(guestId, input.GuestIsolatedDeviceIndex)
	if err != nil {
		return nil, errors.Wrap(err, "GetGuestIsolatedDevice")
	}
	return &hostapi.ContainerDevice{
		Type: dev.Type,
		IsolatedDevice: &hostapi.ContainerIsolatedDevice{
			Id:          isoDev.GetId(),
			Addr:        isoDev.Addr,
			Path:        isoDev.DevicePath,
			CardPath:    isoDev.CardPath,
			DeviceType:  isoDev.DevType,
			SharingMode: isoDev.SharingMode,
			RenderPath:  isoDev.RenderPath,
			MemoryLimit: gdev.DeviceMemorySize,
			SmUtilLimit: gdev.SmUtilLimit,
			Index:       isoDev.Index,
			DeviceMinor: isoDev.DeviceMinor,
			OnlyEnv:     input.OnlyEnv,
			CDI:         input.CDI,
		},
	}, nil
}

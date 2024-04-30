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
	"strings"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	models.RegisterContainerDeviceDriver(newHostDevice())
}

type hostDevice struct {
}

func newHostDevice() models.IContainerDeviceDriver {
	return &hostDevice{}
}

func (h hostDevice) GetType() apis.ContainerDeviceType {
	return apis.CONTAINER_DEVICE_TYPE_HOST
}

func (h hostDevice) ValidatePodCreateData(ctx context.Context, userCred mcclient.TokenCredential, dev *api.ContainerDevice, input *api.ServerCreateInput) error {
	_, err := h.ValidateCreateData(ctx, userCred, nil, dev)
	return err
}

func (h hostDevice) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, pod *models.SGuest, dev *api.ContainerDevice) (*api.ContainerDevice, error) {
	host := dev.Host
	if host == nil {
		return nil, httperrors.NewNotEmptyError("host is nil")
	}
	if host.HostPath == "" {
		return nil, httperrors.NewNotEmptyError("host_path is empty")
	}
	if host.ContainerPath == "" {
		return nil, httperrors.NewNotEmptyError("container_path is empty")
	}
	if host.Permissions == "" {
		return nil, httperrors.NewNotEmptyError("permissions is empty")
	}
	for _, p := range strings.Split(host.Permissions, "") {
		switch p {
		case "r", "w", "m":
		default:
			return nil, httperrors.NewInputParameterError("wrong permission %s", p)
		}
	}
	return dev, nil
}

func (h hostDevice) ToHostDevice(dev *api.ContainerDevice) (*hostapi.ContainerDevice, error) {
	return &hostapi.ContainerDevice{
		Type:          apis.CONTAINER_DEVICE_TYPE_HOST,
		ContainerPath: dev.Host.ContainerPath,
		Permissions:   dev.Host.Permissions,
		Host:          &hostapi.ContainerHostDevice{HostPath: dev.Host.HostPath},
	}, nil
}

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

package hostdrivers

import (
	"context"
	"fmt"

	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SGoogleHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SGoogleHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SGoogleHostDriver) GetHostType() string {
	return api.HOST_TYPE_GOOGLE
}

func (self *SGoogleHostDriver) GetHypervisor() string {
	return api.HYPERVISOR_GOOGLE
}

func (self *SGoogleHostDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_GOOGLE
}

func (self *SGoogleHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	minGB := 10
	maxGB := -1
	switch storage.StorageType {
	case api.STORAGE_GOOGLE_PD_SSD, api.STORAGE_GOOGLE_PD_STANDARD, api.STORAGE_GOOGLE_PD_BALANCED:
		maxGB = 65536
	default:
		return fmt.Errorf("Not support resize %s disk", storage.StorageType)
	}
	if sizeGb < minGB || sizeGb > maxGB {
		return fmt.Errorf("The %s disk size must be in the range of %dG ~ %dGB", storage.StorageType, minGB, maxGB)
	}
	return nil
}

func (self *SGoogleHostDriver) ValidateResetDisk(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, snapshot *models.SSnapshot, guests []models.SGuest, input *api.DiskResetInput) (*api.DiskResetInput, error) {
	for _, guest := range guests {
		if !utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_READY}) {
			return nil, httperrors.NewBadGatewayError("%s reset disk required guest status is running or ready", self.GetHostType())
		}
	}
	return input, nil
}

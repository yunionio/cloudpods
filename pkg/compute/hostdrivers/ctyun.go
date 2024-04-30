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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCtyunHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SCtyunHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SCtyunHostDriver) GetHostType() string {
	return api.HOST_TYPE_CTYUN
}

func (self *SCtyunHostDriver) GetHypervisor() string {
	return api.HYPERVISOR_CTYUN
}

func (self *SCtyunHostDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_CTYUN
}

// 系统盘必须至少40G
func (self *SCtyunHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	switch storage.StorageType {
	case api.STORAGE_CTYUN_SSD, api.STORAGE_CTYUN_SATA, api.STORAGE_CTYUN_SAS:
		if sizeGb < 10 || sizeGb > 32768 {
			return fmt.Errorf("The %s disk size must be in the range of 10G ~ 32768GB", storage.StorageType)
		}
	default:
		return fmt.Errorf("Not support create %s disk", storage.StorageType)
	}

	return nil
}

func (self *SCtyunHostDriver) ValidateResetDisk(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, snapshot *models.SSnapshot, guests []models.SGuest, input *api.DiskResetInput) (*api.DiskResetInput, error) {
	if len(guests) >= 1 {
		return nil, httperrors.NewBadRequestError("Disk must be dettached")
	}
	return input, nil
}

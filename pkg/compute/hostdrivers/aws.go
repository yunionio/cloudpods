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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SAwsHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SAwsHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SAwsHostDriver) GetHostType() string {
	return api.HOST_TYPE_AWS
}

func (self *SAwsHostDriver) GetHypervisor() string {
	return api.HYPERVISOR_AWS
}

func (self *SAwsHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	if storage.StorageType == api.STORAGE_GP2_SSD {
		if sizeGb < 1 || sizeGb > 16384 {
			return fmt.Errorf("The %s disk size must be in the range of 1G ~ 16384GB", storage.StorageType)
		}
	} else if storage.StorageType == api.STORAGE_IO1_SSD {
		if sizeGb < 4 || sizeGb > 16384 {
			return fmt.Errorf("The %s disk size must be in the range of 4G ~ 16384GB", storage.StorageType)
		}
	} else if utils.IsInStringArray(storage.StorageType, []string{api.STORAGE_ST1_HDD, api.STORAGE_SC1_HDD}) {
		if sizeGb < 500 || sizeGb > 16384 {
			return fmt.Errorf("The %s disk size must be in the range of 500G ~ 16384GB", storage.StorageType)
		}
	} else if storage.StorageType == api.STORAGE_STANDARD_HDD {
		if sizeGb < 1 || sizeGb > 1024 {
			return fmt.Errorf("The %s disk size must be in the range of 1G ~ 1024GB", storage.StorageType)
		}
	} else {
		return fmt.Errorf("Not support create %s disk", storage.StorageType)
	}
	return nil
}

func (self *SAwsHostDriver) ValidateResetDisk(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, snapshot *models.SSnapshot, guests []models.SGuest, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewBadRequestError("Aws not support reset disk, you can create new disk with snapshot")
}

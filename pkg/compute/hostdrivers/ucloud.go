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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SUCloudHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SUCloudHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SUCloudHostDriver) GetHostType() string {
	return api.HOST_TYPE_UCLOUD
}

func (self *SUCloudHostDriver) GetHypervisor() string {
	return api.HYPERVISOR_UCLOUD
}

func (self *SUCloudHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	if storage.StorageType == api.STORAGE_UCLOUD_CLOUD_NORMAL {
		if sizeGb < 20 || sizeGb > 8000 {
			return fmt.Errorf("The %s disk size must be in the range of 20G ~ 8000GB", storage.StorageType)
		}
	} else if storage.StorageType == api.STORAGE_UCLOUD_CLOUD_SSD {
		if sizeGb < 20 || sizeGb > 4000 {
			return fmt.Errorf("The %s disk size must be in the range of 20G ~ 4000GB", storage.StorageType)
		}
	} else if storage.StorageType == api.STORAGE_UCLOUD_LOCAL_SSD {
		if sizeGb < 20 || sizeGb > 1000 {
			return fmt.Errorf("The %s disk size must be in the range of 20G ~ 1000GB", storage.StorageType)
		}

		return fmt.Errorf("Not support create/resize %s disk", storage.StorageType)
	} else if storage.StorageType == api.STORAGE_UCLOUD_LOCAL_NORMAL {
		if sizeGb < 20 || sizeGb > 2000 {
			return fmt.Errorf("The %s disk size must be in the range of 20G ~ 2000GB", storage.StorageType)
		}

		return fmt.Errorf("Not support create/resize %s disk", storage.StorageType)
	} else {
		return fmt.Errorf("Not support create %s disk", storage.StorageType)
	}

	return nil
}

func (self *SUCloudHostDriver) ValidateResetDisk(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, snapshot *models.SSnapshot, guests []models.SGuest, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if len(guests) > 0 {
		return nil, httperrors.NewInputParameterError("Ucloud reset disk operation required disk not be attached")
	}
	if disk.DiskType != api.DISK_TYPE_DATA {
		return nil, httperrors.NewInputParameterError("Ucloud only support data disk reset operation")
	}
	return data, nil
}

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
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type SQcloudHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SQcloudHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SQcloudHostDriver) GetHostType() string {
	return api.HOST_TYPE_QCLOUD
}

func (self *SQcloudHostDriver) ValidateAttachStorage(host *models.SHost, storage *models.SStorage, data *jsonutils.JSONDict) error {
	return httperrors.NewUnsupportOperationError("Not support attach storage for %s host", self.GetHostType())
}

func (self *SQcloudHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	if sizeGb%10 != 0 {
		return fmt.Errorf("The disk size must be a multiple of 10Gb")
	}
	if storage.StorageType == api.STORAGE_CLOUD_BASIC {
		if sizeGb < 10 || sizeGb > 16000 {
			return fmt.Errorf("The %s disk size must be in the range of 10 ~ 16000GB", storage.StorageType)
		}
	} else if storage.StorageType == api.STORAGE_CLOUD_PREMIUM {
		if sizeGb < 50 || sizeGb > 16000 {
			return fmt.Errorf("The %s disk size must be in the range of 50 ~ 16000GB", storage.StorageType)
		}
	} else if storage.StorageType == api.STORAGE_CLOUD_SSD {
		if sizeGb < 100 || sizeGb > 16000 {
			return fmt.Errorf("The %s disk size must be in the range of 100 ~ 16000GB", storage.StorageType)
		}
	} else {
		return fmt.Errorf("Not support create %s disk", storage.StorageType)
	}
	return nil
}

func (self *SQcloudHostDriver) RequestDeleteSnapshotWithStorage(ctx context.Context, host *models.SHost, snapshot *models.SSnapshot, task taskman.ITask) error {
	return httperrors.NewNotImplementedError("not implement")
}

func (driver *SQcloudHostDriver) GetStoragecacheQuota(host *models.SHost) int {
	return 10
}

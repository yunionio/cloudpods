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
	"fmt"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SUCloudHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SUCloudHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SUCloudHostDriver) GetHostType() string {
	return models.HOST_TYPE_UCLOUD
}

func (self *SUCloudHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	if storage.StorageType == models.STORAGE_UCLOUD_CLOUD_NORMAL {
		if sizeGb < 20 || sizeGb > 8000 {
			return fmt.Errorf("The %s disk size must be in the range of 20G ~ 8000GB", storage.StorageType)
		}
	} else if storage.StorageType == models.STORAGE_UCLOUD_CLOUD_SSD {
		if sizeGb < 20 || sizeGb > 4000 {
			return fmt.Errorf("The %s disk size must be in the range of 20G ~ 4000GB", storage.StorageType)
		}
	} else {
		return fmt.Errorf("Not support create %s disk", storage.StorageType)
	}

	return nil
}

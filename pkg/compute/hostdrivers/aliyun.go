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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type SAliyunHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SAliyunHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SAliyunHostDriver) GetHostType() string {
	return api.HOST_TYPE_ALIYUN
}

func (self *SAliyunHostDriver) ValidateAttachStorage(host *models.SHost, storage *models.SStorage, data *jsonutils.JSONDict) error {
	return httperrors.NewUnsupportOperationError("Not support attach storage for %s host", self.GetHostType())
}

func (self *SAliyunHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	if utils.IsInStringArray(storage.StorageType, []string{api.STORAGE_CLOUD_EFFICIENCY, api.STORAGE_CLOUD_SSD, api.STORAGE_CLOUD_ESSD}) {
		if sizeGb < 20 || sizeGb > 32768 {
			return fmt.Errorf("The %s disk size must be in the range of 20G ~ 32768GB", storage.StorageType)
		}
	} else if storage.StorageType == api.STORAGE_PUBLIC_CLOUD {
		if sizeGb < 5 || sizeGb > 2000 {
			return fmt.Errorf("The %s disk size must be in the range of 5G ~ 2000GB", storage.StorageType)
		}
	} else {
		return fmt.Errorf("Not support create %s disk", storage.StorageType)
	}
	return nil
}

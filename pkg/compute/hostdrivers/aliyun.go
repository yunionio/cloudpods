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

func (self *SAliyunHostDriver) GetHypervisor() string {
	return api.HYPERVISOR_ALIYUN
}

func (self *SAliyunHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	minGB := -1
	maxGB := -1
	switch storage.StorageType {
	case api.STORAGE_CLOUD_EFFICIENCY, api.STORAGE_CLOUD_SSD, api.STORAGE_CLOUD_ESSD:
		minGB = 20
		maxGB = 32768
	case api.STORAGE_CLOUD_ESSD_PL2:
		minGB = 461
		maxGB = 32768
	case api.STORAGE_CLOUD_ESSD_PL3:
		minGB = 1261
		maxGB = 32768
	case api.STORAGE_PUBLIC_CLOUD:
		minGB = 5
		maxGB = 2000
	default:
		return fmt.Errorf("Not support resize %s disk", storage.StorageType)
	}
	if sizeGb < minGB || sizeGb > maxGB {
		return fmt.Errorf("The %s disk size must be in the range of %dG ~ %dGB", storage.StorageType, minGB, maxGB)
	}
	return nil
}

func (self *SAliyunHostDriver) ValidateResetDisk(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, snapshot *models.SSnapshot, guests []models.SGuest, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	for _, guest := range guests {
		if !utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_READY}) {
			return nil, httperrors.NewBadGatewayError("Aliyun reset disk required guest status is running or read")
		}
	}
	return data, nil
}

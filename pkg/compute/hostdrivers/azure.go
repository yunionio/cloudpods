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
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SAzureHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SAzureHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SAzureHostDriver) GetHostType() string {
	return api.HOST_TYPE_AZURE
}

func (self *SAzureHostDriver) GetHypervisor() string {
	return api.HYPERVISOR_AZURE
}

func (self *SAzureHostDriver) ValidateUpdateDisk(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if data.Contains("name") {
		return nil, httperrors.NewInputParameterError("cannot support change azure disk name")
	}
	return data, nil
}

func (self *SAzureHostDriver) ValidateResetDisk(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, snapshot *models.SSnapshot, guests []models.SGuest, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewBadRequestError("Azure not support reset disk, you can create new disk with snapshot")
}

func (self *SAzureHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	if utils.IsInStringArray(storage.StorageType, []string{api.STORAGE_STANDARD_LRS, api.STORAGE_STANDARDSSD_LRS, api.STORAGE_PREMIUM_LRS}) {
		if sizeGb < 1 || sizeGb > 4095 {
			return fmt.Errorf("The %s disk size must be in the range of 1G ~ 4095GB", storage.StorageType)
		}
	} else {
		return fmt.Errorf("Not support create %s disk", storage.StorageType)
	}
	return nil
}

func (self *SAzureHostDriver) RequestDeleteSnapshotWithStorage(ctx context.Context, host *models.SHost, snapshot *models.SSnapshot, task taskman.ITask) error {
	return httperrors.NewNotImplementedError("not implement")
}

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
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
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

func (self *SQcloudHostDriver) GetHypervisor() string {
	return api.HYPERVISOR_QCLOUD
}

func (self *SQcloudHostDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_QCLOUD
}

func (self *SQcloudHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	if sizeGb%10 != 0 {
		return fmt.Errorf("The disk size must be a multiple of 10Gb")
	}
	min, max := 0, 0
	switch storage.StorageType {
	case api.STORAGE_CLOUD_BASIC:
		min, max = 10, 16000
	case api.STORAGE_CLOUD_PREMIUM:
		min, max = 50, 16000
	case api.STORAGE_CLOUD_SSD:
		min, max = 100, 16000
	case api.STORAGE_CLOUD_HSSD:
		min, max = 20, 320000
	default:
		return fmt.Errorf("Not support create or resize %s disk", storage.StorageType)
	}
	if sizeGb < min || sizeGb > max {
		return fmt.Errorf("The %s disk size must be in the range of %d ~ %dGB", storage.StorageType, min, max)
	}
	return nil
}

func (self *SQcloudHostDriver) ValidateResetDisk(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, snapshot *models.SSnapshot, guests []models.SGuest, input *api.DiskResetInput) (*api.DiskResetInput, error) {
	for _, guest := range guests {
		if !utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_READY}) {
			return nil, httperrors.NewBadGatewayError("Qcloud reset disk required guest status is running or ready")
		}
	}
	return input, nil
}

func (self *SQcloudHostDriver) RequestDeleteSnapshotWithStorage(ctx context.Context, host *models.SHost, snapshot *models.SSnapshot, task taskman.ITask) error {
	return httperrors.NewNotImplementedError("not implement")
}

func (driver *SQcloudHostDriver) GetStoragecacheQuota(host *models.SHost) int {
	return 500
}

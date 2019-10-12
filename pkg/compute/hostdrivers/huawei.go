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

type SHuaweiHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SHuaweiHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SHuaweiHostDriver) GetHostType() string {
	return api.HOST_TYPE_HUAWEI
}

func (self *SHuaweiHostDriver) GetHypervisor() string {
	return api.HYPERVISOR_HUAWEI
}

// 系统盘必须至少40G
func (self *SHuaweiHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	switch storage.StorageType {
	case api.STORAGE_HUAWEI_SSD, api.STORAGE_HUAWEI_SATA, api.STORAGE_HUAWEI_SAS:
		if sizeGb < 10 || sizeGb > 32768 {
			return fmt.Errorf("The %s disk size must be in the range of 10G ~ 32768GB", storage.StorageType)
		}
	default:
		return fmt.Errorf("Not support create %s disk", storage.StorageType)
	}

	return nil
}

func (self *SHuaweiHostDriver) ValidateResetDisk(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if disk.Status != api.DISK_READY {
		return nil, httperrors.NewInvalidStatusError("Cannot reset disk in status %s", disk.Status)
	}
	snapshotId, err := data.GetString("snapshot_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("snapshot_id")
	}
	guests := disk.GetGuests()
	if len(guests) >= 1 {
		return nil, httperrors.NewBadRequestError("Disk must be dettached")
	}

	iSnapshot, err := models.SnapshotManager.FetchById(snapshotId)
	if err != nil {
		return nil, httperrors.NewNotFoundError("Snapshot %s not found", snapshotId)
	}
	snapshot := iSnapshot.(*models.SSnapshot)
	if snapshot.Status != api.SNAPSHOT_READY {
		return nil, httperrors.NewBadRequestError("Cannot reset disk with snapshot in status %s", snapshot.Status)
	} else if snapshot.DiskId != disk.Id {
		return nil, httperrors.NewBadRequestError("Cannot reset disk %s,Snapshot is belong to disk %s", disk.Id, snapshot.DiskId)
	}

	return data, nil
}

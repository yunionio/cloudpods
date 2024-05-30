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

package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type IHostDriver interface {
	GetHostType() string
	GetProvider() string
	GetHypervisor() string

	RequestBaremetalUnmaintence(ctx context.Context, userCred mcclient.TokenCredential, baremetal *SHost, task taskman.ITask) error
	RequestBaremetalMaintence(ctx context.Context, userCred mcclient.TokenCredential, baremetal *SHost, task taskman.ITask) error
	RequestSyncBaremetalHostStatus(ctx context.Context, userCred mcclient.TokenCredential, baremetal *SHost, task taskman.ITask) error
	RequestSyncBaremetalHostConfig(ctx context.Context, userCred mcclient.TokenCredential, baremetal *SHost, task taskman.ITask) error

	CheckAndSetCacheImage(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, storagecache *SStoragecache, task taskman.ITask) error
	RequestUncacheImage(ctx context.Context, host *SHost, storageCache *SStoragecache, task taskman.ITask, deactivateImage bool) error

	ValidateUpdateDisk(ctx context.Context, userCred mcclient.TokenCredential, input *api.DiskUpdateInput) (*api.DiskUpdateInput, error)
	ValidateResetDisk(ctx context.Context, userCred mcclient.TokenCredential, disk *SDisk, snapshot *SSnapshot, guests []SGuest, input *api.DiskResetInput) (*api.DiskResetInput, error)
	ValidateDiskSize(storage *SStorage, sizeGb int) error
	RequestPrepareSaveDiskOnHost(ctx context.Context, host *SHost, disk *SDisk, imageId string, task taskman.ITask) error
	RequestSaveUploadImageOnHost(ctx context.Context, host *SHost, disk *SDisk, imageId string, task taskman.ITask, data jsonutils.JSONObject) error

	// create disk
	RequestAllocateDiskOnStorage(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, storage *SStorage, disk *SDisk, task taskman.ITask, input api.DiskAllocateInput) error
	RequestRebuildDiskOnStorage(ctx context.Context, host *SHost, storage *SStorage, disk *SDisk, task taskman.ITask, input api.DiskAllocateInput) error

	// delete disk
	RequestDeallocateDiskOnHost(ctx context.Context, host *SHost, storage *SStorage, disk *SDisk, cleanSnapshots bool, task taskman.ITask) error
	RequestDeallocateBackupDiskOnHost(ctx context.Context, host *SHost, storage *SStorage, disk *SDisk, task taskman.ITask) error

	// resize disk
	RequestResizeDiskOnHost(ctx context.Context, host *SHost, storage *SStorage, disk *SDisk, size int64, task taskman.ITask) error
	RequestDiskSrcMigratePrepare(ctx context.Context, host *SHost, disk *SDisk, task taskman.ITask) (jsonutils.JSONObject, error)
	RequestDiskMigrate(ctx context.Context, targetHost *SHost, targetStorage *SStorage, disk *SDisk, task taskman.ITask, body *jsonutils.JSONDict) error

	RequestDeleteSnapshotsWithStorage(ctx context.Context, host *SHost, snapshot *SSnapshot, task taskman.ITask, snapshotIds []string) error
	RequestDeleteSnapshotWithoutGuest(ctx context.Context, host *SHost, snapshot *SSnapshot, params *jsonutils.JSONDict, task taskman.ITask) error
	RequestResetDisk(ctx context.Context, host *SHost, disk *SDisk, params *jsonutils.JSONDict, task taskman.ITask) error
	RequestCleanUpDiskSnapshots(ctx context.Context, host *SHost, disk *SDisk, params *jsonutils.JSONDict, task taskman.ITask) error
	PrepareConvert(host *SHost, image, raid string, data jsonutils.JSONObject) (*api.ServerCreateInput, error)
	PrepareUnconvert(host *SHost) error
	FinishUnconvert(ctx context.Context, userCred mcclient.TokenCredential, host *SHost) error
	FinishConvert(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, guest *SGuest, hostType string) error
	ConvertFailed(host *SHost) error
	GetRaidScheme(host *SHost, raid string) (string, error)

	IsReachStoragecacheCapacityLimit(host *SHost, cachedImages []SCachedimage) bool
	GetStoragecacheQuota(host *SHost) int

	ValidateAttachStorage(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, storage *SStorage, input api.HostStorageCreateInput) (api.HostStorageCreateInput, error)
	RequestAttachStorage(ctx context.Context, hoststorage *SHoststorage, host *SHost, storage *SStorage, task taskman.ITask) error
	RequestDetachStorage(ctx context.Context, host *SHost, storage *SStorage, task taskman.ITask) error
	RequestSyncOnHost(ctx context.Context, host *SHost, task taskman.ITask) error
	RequestProbeIsolatedDevices(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, input jsonutils.JSONObject) (*jsonutils.JSONArray, error)
}

var hostDrivers map[string]IHostDriver

func init() {
	hostDrivers = make(map[string]IHostDriver)
}

func RegisterHostDriver(driver IHostDriver) {
	key := fmt.Sprintf("%s-%s", driver.GetHostType(), driver.GetProvider())
	hostDrivers[key] = driver
}

func GetHostDriver(hostType, provider string) (IHostDriver, error) {
	key := fmt.Sprintf("%s-%s", hostType, provider)
	driver, ok := hostDrivers[key]
	if ok {
		return driver, nil
	}
	return nil, errors.Wrapf(errors.ErrNotFound, "host type: %s provider: %s", hostType, provider)
}

func Hypervisors2HostTypes(hypervisors []string) []string {
	ret := []string{}
	for _, driver := range hostDrivers {
		if !utils.IsInStringArray(driver.GetHypervisor(), hypervisors) {
			continue
		}
		if utils.IsInStringArray(driver.GetHostType(), ret) {
			continue
		}
		ret = append(ret, driver.GetHostType())
	}
	return ret
}

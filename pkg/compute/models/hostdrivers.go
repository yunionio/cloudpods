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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type IHostDriver interface {
	GetHostType() string
	GetHypervisor() string

	CheckAndSetCacheImage(ctx context.Context, host *SHost, storagecache *SStoragecache, task taskman.ITask) error
	RequestUncacheImage(ctx context.Context, host *SHost, storageCache *SStoragecache, task taskman.ITask) error

	ValidateUpdateDisk(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	ValidateResetDisk(ctx context.Context, userCred mcclient.TokenCredential, disk *SDisk, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	ValidateDiskSize(storage *SStorage, sizeGb int) error
	RequestPrepareSaveDiskOnHost(ctx context.Context, host *SHost, disk *SDisk, imageId string, task taskman.ITask) error
	RequestSaveUploadImageOnHost(ctx context.Context, host *SHost, disk *SDisk, imageId string, task taskman.ITask, data jsonutils.JSONObject) error

	RequestAllocateDiskOnStorage(ctx context.Context, host *SHost, storage *SStorage, disk *SDisk, task taskman.ITask, content *jsonutils.JSONDict) error
	RequestRebuildDiskOnStorage(ctx context.Context, host *SHost, storage *SStorage, disk *SDisk, task taskman.ITask, content *jsonutils.JSONDict) error

	RequestDeallocateDiskOnHost(ctx context.Context, host *SHost, storage *SStorage, disk *SDisk, task taskman.ITask) error
	RequestDeallocateBackupDiskOnHost(ctx context.Context, host *SHost, storage *SStorage, disk *SDisk, task taskman.ITask) error

	RequestResizeDiskOnHost(ctx context.Context, host *SHost, storage *SStorage, disk *SDisk, size int64, task taskman.ITask) error

	RequestDeleteSnapshotsWithStorage(ctx context.Context, host *SHost, snapshot *SSnapshot, task taskman.ITask) error
	RequestResetDisk(ctx context.Context, host *SHost, disk *SDisk, params *jsonutils.JSONDict, task taskman.ITask) error
	RequestCleanUpDiskSnapshots(ctx context.Context, host *SHost, disk *SDisk, params *jsonutils.JSONDict, task taskman.ITask) error
	PrepareConvert(host *SHost, image, raid string, data jsonutils.JSONObject) (*api.ServerCreateInput, error)
	PrepareUnconvert(host *SHost) error
	FinishUnconvert(ctx context.Context, userCred mcclient.TokenCredential, host *SHost) error
	FinishConvert(userCred mcclient.TokenCredential, host *SHost, guest *SGuest, hostType string) error
	ConvertFailed(host *SHost) error
	GetRaidScheme(host *SHost, raid string) (string, error)

	IsReachStoragecacheCapacityLimit(host *SHost, cachedImages []SCachedimage) bool
	GetStoragecacheQuota(host *SHost) int

	ValidateAttachStorage(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, storage *SStorage, data *jsonutils.JSONDict) error
	RequestAttachStorage(ctx context.Context, hoststorage *SHoststorage, host *SHost, storage *SStorage, task taskman.ITask) error
	RequestDetachStorage(ctx context.Context, host *SHost, storage *SStorage, task taskman.ITask) error
}

var hostDrivers map[string]IHostDriver

func init() {
	hostDrivers = make(map[string]IHostDriver)
}

func RegisterHostDriver(driver IHostDriver) {
	hostDrivers[driver.GetHostType()] = driver
}

func GetHostDriver(hostType string) IHostDriver {
	driver, ok := hostDrivers[hostType]
	if ok {
		return driver
	} else {
		log.Fatalf("Unsupported hostType %s", hostType)
		return nil
	}
}

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
	"sync/atomic"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

var (
	cleanupDuplicateExternalRunning int32
	cleanupDuplicateExternalLastRun int64 // unix nano，配合配置间隔热更新
)

// CleanupDuplicateExternalResources 定时清理同步竞态产生的重复虚拟机/磁盘：
// 同一 cloudprovider(manager_id) 下 external_id 相同，且同时存在 source=local 与 source=cloud 时，
// 保留 local，仅删除 cloud 侧的数据库记录（RealDelete/purge，不调用云上删除）。
// cron 每小时触发一次；是否执行、实际间隔与 batch 均读当前 options，改配置无需重启。
func CleanupDuplicateExternalResources(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if !options.Options.EnableAutoCleanupDuplicateExternalResources {
		return
	}
	intervalHours := options.Options.AutoCleanupDuplicateExternalIntervalHours
	if intervalHours <= 0 {
		return
	}
	now := time.Now()
	last := atomic.LoadInt64(&cleanupDuplicateExternalLastRun)
	if last > 0 && now.Sub(time.Unix(0, last)) < time.Duration(intervalHours)*time.Hour {
		return
	}
	if !atomic.CompareAndSwapInt32(&cleanupDuplicateExternalRunning, 0, 1) {
		log.Warningf("Previous CleanupDuplicateExternalResources still running, skip")
		return
	}
	defer atomic.StoreInt32(&cleanupDuplicateExternalRunning, 0)
	atomic.StoreInt64(&cleanupDuplicateExternalLastRun, now.UnixNano())

	batchSize := options.Options.AutoCleanupDuplicateExternalMaxCleanBatchSize
	if batchSize <= 0 {
		batchSize = 50
	}

	// 先清重复虚机（detach 其云侧磁盘关联），再清未挂载的重复磁盘
	guestCnt, err := GuestManager.cleanupDuplicateExternalGuests(ctx, userCred, batchSize)
	if err != nil {
		log.Errorf("cleanupDuplicateExternalGuests: %v", err)
	}
	diskCnt, err := DiskManager.cleanupDuplicateExternalDisks(ctx, userCred, batchSize)
	if err != nil {
		log.Errorf("cleanupDuplicateExternalDisks: %v", err)
	}
	if guestCnt > 0 || diskCnt > 0 {
		log.Infof("CleanupDuplicateExternalResources purged %d cloud guests, %d cloud disks", guestCnt, diskCnt)
	}
}

func (manager *SGuestManager) cleanupDuplicateExternalGuests(
	ctx context.Context, userCred mcclient.TokenCredential, batchSize int,
) (int, error) {
	guests, err := manager.getDuplicateCloudGuests(batchSize)
	if err != nil {
		return 0, errors.Wrap(err, "getDuplicateCloudGuests")
	}
	cleaned := 0
	for i := range guests {
		guest := &guests[i]
		if guest.Freezed {
			log.Infof("skip freezed duplicate cloud guest %s(%s)", guest.Name, guest.Id)
			continue
		}
		if guest.Status == api.VM_RUNNING {
			log.Infof("skip running duplicate cloud guest %s(%s)", guest.Name, guest.Id)
			continue
		}

		lockman.LockObject(ctx, guest)
		err := func() error {
			defer lockman.ReleaseObject(ctx, guest)
			// 再次确认仍存在同 manager+external_id 的 local 记录
			if !manager.hasLocalTwinGuest(guest) {
				return nil
			}
			db.OpsLog.LogEvent(guest, db.ACT_DELETE, "auto cleanup duplicate cloud guest with same external_id, keep local", userCred)
			logclient.AddSimpleActionLog(guest, logclient.ACT_DELETE, "auto cleanup duplicate cloud guest (same external_id, keep local)", userCred, true)
			return guest.RealDelete(ctx, userCred)
		}()
		if err != nil {
			log.Errorf("RealDelete duplicate cloud guest %s(%s): %v", guest.Name, guest.Id, err)
			db.OpsLog.LogEvent(guest, db.ACT_DELETE_FAIL, err.Error(), userCred)
			continue
		}
		cleaned++
	}
	return cleaned, nil
}

func (manager *SGuestManager) getDuplicateCloudGuests(batchSize int) ([]SGuest, error) {
	hosts := HostManager.Query().SubQuery()

	localGuests := manager.Query("external_id", "host_id").
		Equals("source", apis.EXTERNAL_RESOURCE_SOURCE_LOCAL).
		IsNotEmpty("external_id").
		IsFalse("pending_deleted").
		SubQuery()
	localWithMgr := localGuests.Query(
		localGuests.Field("external_id"),
		hosts.Field("manager_id"),
	).Join(hosts, sqlchemy.Equals(localGuests.Field("host_id"), hosts.Field("id"))).
		Filter(sqlchemy.IsNotEmpty(hosts.Field("manager_id"))).
		SubQuery()

	q := manager.Query().
		Equals("source", apis.EXTERNAL_RESOURCE_SOURCE_CLOUD).
		IsNotEmpty("external_id").
		IsFalse("pending_deleted")
	q = q.Join(hosts, sqlchemy.Equals(q.Field("host_id"), hosts.Field("id")))
	q = q.Join(localWithMgr, sqlchemy.AND(
		sqlchemy.Equals(q.Field("external_id"), localWithMgr.Field("external_id")),
		sqlchemy.Equals(hosts.Field("manager_id"), localWithMgr.Field("manager_id")),
	))
	q = q.Filter(sqlchemy.IsNotEmpty(hosts.Field("manager_id")))
	q = q.Asc("created_at").Limit(batchSize)

	guests := make([]SGuest, 0)
	err := db.FetchModelObjects(manager, q, &guests)
	if err != nil {
		return nil, err
	}
	return guests, nil
}

func (manager *SGuestManager) hasLocalTwinGuest(guest *SGuest) bool {
	host, err := guest.GetHost()
	if err != nil || host == nil || len(host.ManagerId) == 0 || len(guest.ExternalId) == 0 {
		return false
	}
	hosts := HostManager.Query().SubQuery()
	q := manager.Query().
		Equals("source", apis.EXTERNAL_RESOURCE_SOURCE_LOCAL).
		Equals("external_id", guest.ExternalId).
		IsFalse("pending_deleted").
		NotEquals("id", guest.Id)
	q = q.Join(hosts, sqlchemy.Equals(q.Field("host_id"), hosts.Field("id")))
	q = q.Filter(sqlchemy.Equals(hosts.Field("manager_id"), host.ManagerId))
	cnt, err := q.CountWithError()
	if err != nil {
		log.Errorf("hasLocalTwinGuest count %s: %v", guest.Id, err)
		return false
	}
	return cnt > 0
}

func (manager *SDiskManager) cleanupDuplicateExternalDisks(
	ctx context.Context, userCred mcclient.TokenCredential, batchSize int,
) (int, error) {
	disks, err := manager.getDuplicateCloudDisks(batchSize)
	if err != nil {
		return 0, errors.Wrap(err, "getDuplicateCloudDisks")
	}
	cleaned := 0
	for i := range disks {
		disk := &disks[i]
		if disk.Freezed {
			log.Infof("skip freezed duplicate cloud disk %s(%s)", disk.Name, disk.Id)
			continue
		}
		cnt, err := disk.GetGuestDiskCount()
		if err != nil {
			log.Errorf("GetGuestDiskCount disk %s: %v", disk.Id, err)
			continue
		}
		if cnt > 0 {
			log.Infof("skip attached duplicate cloud disk %s(%s) guest_count=%d", disk.Name, disk.Id, cnt)
			continue
		}

		lockman.LockObject(ctx, disk)
		err = func() error {
			defer lockman.ReleaseObject(ctx, disk)
			if !manager.hasLocalTwinDisk(disk) {
				return nil
			}
			// 清理前再确认未挂载
			cnt, err := disk.GetGuestDiskCount()
			if err != nil {
				return err
			}
			if cnt > 0 {
				return nil
			}
			db.OpsLog.LogEvent(disk, db.ACT_DELETE, "auto cleanup duplicate cloud disk with same external_id, keep local", userCred)
			logclient.AddSimpleActionLog(disk, logclient.ACT_DELETE, "auto cleanup duplicate cloud disk (same external_id, keep local)", userCred, true)
			return disk.RealDelete(ctx, userCred)
		}()
		if err != nil {
			log.Errorf("RealDelete duplicate cloud disk %s(%s): %v", disk.Name, disk.Id, err)
			db.OpsLog.LogEvent(disk, db.ACT_DELETE_FAIL, err.Error(), userCred)
			continue
		}
		cleaned++
	}
	return cleaned, nil
}

func (manager *SDiskManager) getDuplicateCloudDisks(batchSize int) ([]SDisk, error) {
	storages := StorageManager.Query().SubQuery()

	localDisks := manager.Query("external_id", "storage_id").
		Equals("source", apis.EXTERNAL_RESOURCE_SOURCE_LOCAL).
		IsNotEmpty("external_id").
		IsFalse("pending_deleted").
		SubQuery()
	localWithMgr := localDisks.Query(
		localDisks.Field("external_id"),
		storages.Field("manager_id"),
	).Join(storages, sqlchemy.Equals(localDisks.Field("storage_id"), storages.Field("id"))).
		Filter(sqlchemy.IsNotEmpty(storages.Field("manager_id"))).
		SubQuery()

	q := manager.Query().
		Equals("source", apis.EXTERNAL_RESOURCE_SOURCE_CLOUD).
		IsNotEmpty("external_id").
		IsFalse("pending_deleted")
	q = q.Join(storages, sqlchemy.Equals(q.Field("storage_id"), storages.Field("id")))
	q = q.Join(localWithMgr, sqlchemy.AND(
		sqlchemy.Equals(q.Field("external_id"), localWithMgr.Field("external_id")),
		sqlchemy.Equals(storages.Field("manager_id"), localWithMgr.Field("manager_id")),
	))
	q = q.Filter(sqlchemy.IsNotEmpty(storages.Field("manager_id")))
	q = q.Asc("created_at").Limit(batchSize)

	disks := make([]SDisk, 0)
	err := db.FetchModelObjects(manager, q, &disks)
	if err != nil {
		return nil, err
	}
	return disks, nil
}

func (manager *SDiskManager) hasLocalTwinDisk(disk *SDisk) bool {
	storage, err := disk.GetStorage()
	if err != nil || storage == nil || len(storage.ManagerId) == 0 || len(disk.ExternalId) == 0 {
		return false
	}
	storages := StorageManager.Query().SubQuery()
	q := manager.Query().
		Equals("source", apis.EXTERNAL_RESOURCE_SOURCE_LOCAL).
		Equals("external_id", disk.ExternalId).
		IsFalse("pending_deleted").
		NotEquals("id", disk.Id)
	q = q.Join(storages, sqlchemy.Equals(q.Field("storage_id"), storages.Field("id")))
	q = q.Filter(sqlchemy.Equals(storages.Field("manager_id"), storage.ManagerId))
	cnt, err := q.CountWithError()
	if err != nil {
		log.Errorf("hasLocalTwinDisk count %s: %v", disk.Id, err)
		return false
	}
	return cnt > 0
}

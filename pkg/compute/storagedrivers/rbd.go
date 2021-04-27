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

package storagedrivers

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type SRbdStorageDriver struct {
	SBaseStorageDriver
}

func init() {
	driver := SRbdStorageDriver{}
	models.RegisterStorageDriver(&driver)
}

func (self *SRbdStorageDriver) GetStorageType() string {
	return api.STORAGE_RBD
}

func (self *SRbdStorageDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.StorageCreateInput) error {
	input.StorageConf = jsonutils.NewDict()
	if len(input.RbdMonHost) == 0 {
		return httperrors.NewMissingParameterError("rbd_mon_host")
	}
	input.MonHost = strings.ReplaceAll(input.RbdMonHost, " ", "")

	if len(input.RbdPool) == 0 {
		return httperrors.NewMissingParameterError("rbd_pool")
	}
	input.Pool = strings.Trim(input.RbdPool, " ")
	input.Key = strings.Trim(input.RbdKey, " ")

	input.RadosMonOpTimeout = input.RbdRadosMonOpTimeout
	if input.RadosMonOpTimeout <= 0 {
		input.RadosMonOpTimeout = api.RBD_DEFAULT_MON_TIMEOUT
	}
	input.RadosOsdOpTimeout = input.RbdRadosOsdOpTimeout
	if input.RadosOsdOpTimeout <= 0 {
		input.RadosOsdOpTimeout = api.RBD_DEFAULT_OSD_TIMEOUT
	}
	input.ClientMountTimeout = input.RbdClientMountTimeout
	if input.ClientMountTimeout <= 0 {
		input.ClientMountTimeout = api.RBD_DEFAULT_MOUNT_TIMEOUT
	}

	storages := []models.SStorage{}
	q := models.StorageManager.Query().Equals("storage_type", api.STORAGE_RBD)
	err := db.FetchModelObjects(models.StorageManager, q, &storages)
	if err != nil {
		return httperrors.NewGeneralError(err)
	}

	for i := 0; i < len(storages); i++ {
		host, _ := storages[i].StorageConf.GetString("mon_host")
		pool, _ := storages[i].StorageConf.GetString("pool")
		if input.MonHost == host && input.Pool == pool {
			return httperrors.NewDuplicateResourceError("This RBD Storage[%s/%s] has already exist", storages[i].Name, input.Pool)
		}
	}

	input.StorageConf.Update(
		jsonutils.Marshal(map[string]interface{}{
			"mon_host":             input.MonHost,
			"pool":                 input.Pool,
			"key":                  input.Key,
			"rados_mon_op_timeout": input.RadosMonOpTimeout,
			"rados_osd_op_timeout": input.RadosOsdOpTimeout,
			"client_mount_timeout": input.ClientMountTimeout,
		}))
	return nil
}

func (self *SRbdStorageDriver) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, input api.StorageUpdateInput) (api.StorageUpdateInput, error) {
	for k, v := range map[string]int{
		"rados_mon_op_timeout": input.RbdRadosMonOpTimeout,
		"rados_osd_op_timeout": input.RbdRadosOsdOpTimeout,
		"client_mount_timeout": input.RbdClientMountTimeout} {
		if v > 0 {
			input.StorageConf.Set(k, jsonutils.NewInt(int64(v)))
			input.UpdateStorageConf = true
		}
	}

	if len(input.RbdKey) > 0 {
		input.StorageConf.Set("key", jsonutils.NewString(strings.Trim(input.RbdKey, " ")))
		input.UpdateStorageConf = true
	}

	return input, nil
}

func (self *SRbdStorageDriver) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, storage *models.SStorage, data jsonutils.JSONObject) {
	storages := []models.SStorage{}
	q := models.StorageManager.Query().Equals("storage_type", api.STORAGE_RBD)
	if err := db.FetchModelObjects(models.StorageManager, q, &storages); err != nil {
		log.Errorf("fetch storages error: %v", err)
		return
	}
	newRbdHost, _ := data.GetString("rbd_mon_host")
	newRbdKey, _ := data.GetString("rbd_key")
	for i := 0; i < len(storages); i++ {
		rbdHost, _ := storages[i].StorageConf.GetString("mon_host")
		rbdKey, _ := storages[i].StorageConf.GetString("key")
		if newRbdHost == rbdHost && newRbdKey == rbdKey {
			_, err := db.Update(storage, func() error {
				storage.StoragecacheId = storages[i].StoragecacheId
				return nil
			})
			if err != nil {
				log.Errorf("Update storagecacheId error: %v", err)
				return
			}
		}
	}
	if len(storage.StoragecacheId) == 0 {
		sc := &models.SStoragecache{}
		sc.SetModelManager(models.StoragecacheManager, sc)
		sc.Name = fmt.Sprintf("imagecache-%s", storage.Id)
		pool, _ := data.GetString("rbd_pool")
		sc.Path = fmt.Sprintf("rbd:%s", pool)
		if err := models.StoragecacheManager.TableSpec().Insert(ctx, sc); err != nil {
			log.Errorf("insert storagecache for storage %s error: %v", storage.Name, err)
			return
		}
		_, err := db.Update(storage, func() error {
			storage.StoragecacheId = sc.Id
			return nil
		})
		if err != nil {
			log.Errorf("update storagecache info for storage %s error: %v", storage.Name, err)
		}
	}
}

func (self *SRbdStorageDriver) DoStorageUpdateTask(ctx context.Context, userCred mcclient.TokenCredential, storage *models.SStorage, task taskman.ITask) error {
	subtask, err := taskman.TaskManager.NewTask(ctx, "RbdStorageUpdateTask", storage, task.GetUserCred(), task.GetParams(), task.GetTaskId(), "", nil)
	if err != nil {
		return err
	}
	subtask.ScheduleRun(nil)
	return nil
}

func (self *SRbdStorageDriver) ValidateSnapshotDelete(ctx context.Context, snapshot *models.SSnapshot) error {
	return nil
}

func (self *SRbdStorageDriver) ValidateCreateSnapshotData(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, input *api.SnapshotCreateInput) error {
	return nil
}

func (self *SRbdStorageDriver) RequestCreateSnapshot(ctx context.Context, snapshot *models.SSnapshot, task taskman.ITask) error {
	disk, err := snapshot.GetDisk()
	if err != nil {
		return errors.Wrap(err, "snapshot get disk")
	}
	storage := snapshot.GetStorage()
	host := storage.GetMasterHost()
	url := fmt.Sprintf("%s/disks/%s/snapshot/%s", host.ManagerUri, storage.Id, disk.Id)
	header := task.GetTaskRequestHeader()
	params := jsonutils.NewDict()
	params.Set("snapshot_id", jsonutils.NewString(snapshot.Id))
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, params, false)
	if err != nil {
		return errors.Wrap(err, "request create snapshot")
	}
	return nil
}

func (self *SRbdStorageDriver) RequestDeleteSnapshot(ctx context.Context, snapshot *models.SSnapshot, task taskman.ITask) error {
	storage := snapshot.GetStorage()
	host := storage.GetMasterHost()
	if host == nil {
		return errors.Errorf("storage %s can't get master host", storage.Id)
	}
	url := fmt.Sprintf("%s/disks/%s/delete-snapshot/%s", host.ManagerUri, storage.Id, snapshot.DiskId)
	header := task.GetTaskRequestHeader()
	params := jsonutils.NewDict()
	params.Set("snapshot_id", jsonutils.NewString(snapshot.Id))
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, params, false)
	if err != nil {
		return errors.Wrap(err, "request delete snapshot")
	}
	return nil
}

func (self *SRbdStorageDriver) SnapshotIsOutOfChain(disk *models.SDisk) bool {
	return true
}

func (self *SRbdStorageDriver) OnDiskReset(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, snapshot *models.SSnapshot, data jsonutils.JSONObject) error {
	return nil
}

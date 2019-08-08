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

func (self *SRbdStorageDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	conf := jsonutils.NewDict()
	for _, v := range []string{"rbd_mon_host", "rbd_pool"} {
		if !data.Contains(v) {
			return nil, httperrors.NewMissingParameterError(v)
		}
		value, _ := data.GetString(v)
		conf.Add(jsonutils.NewString(value), strings.TrimPrefix(v, "rbd_"))
	}
	if key, _ := data.GetString("rbd_key"); len(key) > 0 {
		conf.Add(jsonutils.NewString(key), "key")
	}

	for k, v := range map[string]int64{
		"rbd_rados_mon_op_timeout": api.RBD_DEFAULT_MON_TIMEOUT,
		"rbd_rados_osd_op_timeout": api.RBD_DEFAULT_OSD_TIMEOUT,
		"rbd_client_mount_timeout": api.RBD_DEFAULT_MOUNT_TIMEOUT,
	} {
		if timeout, _ := data.Int(k); timeout > 0 {
			conf.Add(jsonutils.NewInt(timeout), strings.TrimPrefix(k, "rbd_"))
		} else {
			conf.Add(jsonutils.NewInt(v), strings.TrimPrefix(k, "rbd_"))
		}
	}

	storages := []models.SStorage{}
	q := models.StorageManager.Query().Equals("storage_type", api.STORAGE_RBD)
	if err := db.FetchModelObjects(models.StorageManager, q, &storages); err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	inputHost, _ := conf.GetString("mon_host")
	inputPool, _ := conf.GetString("pool")
	for i := 0; i < len(storages); i++ {
		host, _ := storages[i].StorageConf.GetString("mon_host")
		pool, _ := storages[i].StorageConf.GetString("pool")
		if inputHost == host && inputPool == pool {
			return nil, httperrors.NewDuplicateResourceError("This RBD Storage[%s/%s] has already exist", storages[i].Name, inputPool)
		}
	}

	data.Set("storage_conf", conf)

	return data, nil
}

func (self *SRbdStorageDriver) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, storage *models.SStorage) (*jsonutils.JSONDict, error) {
	conf, ok := storage.StorageConf.(*jsonutils.JSONDict)
	if !ok {
		conf = jsonutils.NewDict()
	}
	data.Set("update_storage_conf", jsonutils.JSONFalse)
	for _, k := range []string{"rbd_rados_mon_op_timeout", "rbd_rados_osd_op_timeout", "rbd_client_mount_timeout"} {
		if timeout, _ := data.Int(k); timeout > 0 {
			conf.Set(strings.TrimPrefix(k, "rbd_"), jsonutils.NewInt(timeout))
			data.Set("update_storage_conf", jsonutils.JSONTrue)
		}
	}

	if key, _ := data.GetString("rbd_key"); len(key) > 0 {
		conf.Set("key", jsonutils.NewString(key))
		data.Set("update_storage_conf", jsonutils.JSONTrue)
	}

	if update, _ := data.Bool("update_storage_conf"); update {
		_, err := storage.GetModelManager().TableSpec().Update(storage, func() error {
			storage.StorageConf = conf
			return nil
		})
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
	}

	return data, nil
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
		if err := models.StoragecacheManager.TableSpec().Insert(sc); err != nil {
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

func (self *SRbdStorageDriver) ValidateSnapshotCreate(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, data *jsonutils.JSONDict) error {
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
	disk, err := snapshot.GetDisk()
	if err != nil {
		return errors.Wrap(err, "snapshot get disk")
	}
	storage := snapshot.GetStorage()
	host := storage.GetMasterHost()
	url := fmt.Sprintf("%s/disks/%s/delete-snapshot/%s", host.ManagerUri, storage.Id, disk.Id)
	header := task.GetTaskRequestHeader()
	params := jsonutils.NewDict()
	params.Set("snapshot_id", jsonutils.NewString(snapshot.Id))
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, params, false)
	if err != nil {
		return errors.Wrap(err, "request create snapshot")
	}
	return nil
}

func (self *SRbdStorageDriver) SnapshotIsOutOfChain(disk *models.SDisk) bool {
	return true
}

func (self *SRbdStorageDriver) OnDiskReset(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, snapshot *models.SSnapshot, data jsonutils.JSONObject) error {
	return nil
}

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

package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type HostStorageDetachTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(HostStorageDetachTask{})
}

func (self *HostStorageDetachTask) taskFail(ctx context.Context, host *models.SHost, reason error) {
	var hoststorage = new(models.SHoststorage)
	storageId, _ := self.GetParams().GetString("storage_id")
	err := models.HoststorageManager.Query().Equals("host_id", host.Id).Equals("storage_id", storageId).First(hoststorage)
	if err == nil {
		storage := hoststorage.GetStorage()
		// note := fmt.Sprintf("detach host %s failed: %s", host.Name, reason)
		db.OpsLog.LogEvent(storage, db.ACT_DETACH_FAIL, reason, self.GetUserCred())
		logclient.AddActionLogWithContext(ctx, storage, logclient.ACT_DETACH_HOST, reason, self.GetUserCred(), false)
	}
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *HostStorageDetachTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	host := obj.(*models.SHost)
	storageId, _ := self.GetParams().GetString("storage_id")
	_storage, err := models.StorageManager.FetchById(storageId)
	if err != nil {
		self.taskFail(ctx, host, errors.Wrapf(err, "FetchById %s", storageId))
		return
	}
	storage := _storage.(*models.SStorage)
	self.SetStage("OnDetachStorageComplete", nil)
	driver, err := host.GetHostDriver()
	if err != nil {
		self.taskFail(ctx, host, errors.Wrapf(err, "GetHostDriver"))
		return
	}
	err = driver.RequestDetachStorage(ctx, host, storage, self)
	if err != nil {
		self.taskFail(ctx, host, errors.Wrapf(err, "RequestDetachStorage"))
	}
}

func (self *HostStorageDetachTask) OnDetachStorageComplete(ctx context.Context, host *models.SHost, data jsonutils.JSONObject) {
	storageId, _ := self.GetParams().GetString("storage_id")
	storage := models.StorageManager.FetchStorageById(storageId)
	db.OpsLog.LogEvent(storage, db.ACT_DETACH, "", self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, storage, logclient.ACT_DETACH_HOST,
		fmt.Sprintf("Detach host %s success", host.Name), self.GetUserCred(), true)
	self.SetStageComplete(ctx, nil)
	storage.SyncStatusWithHosts(ctx)
	host.ClearSchedDescCache()
}

func (self *HostStorageDetachTask) OnDetachStorageCompleteFailed(ctx context.Context, host *models.SHost, reason jsonutils.JSONObject) {
	self.taskFail(ctx, host, errors.Errorf(reason.String()))
}

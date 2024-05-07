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
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	taskman.RegisterTask(StorageUpdateTask{})
	taskman.RegisterTask(RbdStorageUpdateTask{})
}

type StorageUpdateTask struct {
	taskman.STask
}

func (self *StorageUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStage("OnStorageUpdate", nil)
	storage := obj.(*models.SStorage)
	driver := models.GetStorageDriver(storage.StorageType)
	if driver != nil {
		err := driver.DoStorageUpdateTask(ctx, self.UserCred, storage, self)
		if err != nil {
			self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		}
	}
	self.SetStageComplete(ctx, nil)
}

func (self *StorageUpdateTask) OnStorageUpdate(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *StorageUpdateTask) OnStorageUpdateFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}

type RbdStorageUpdateTask struct {
	taskman.STask
}

func (self *RbdStorageUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	storage := obj.(*models.SStorage)
	hosts := storage.GetAllAttachingHosts()

	for _, host := range hosts {
		log.Infof("Updata rbd Storage [%s] on host %s ...", storage.Name, host.Name)
		url := fmt.Sprintf("%s/storages/update", host.ManagerUri)
		headers := mcclient.GetTokenHeaders(self.GetUserCred())
		body := jsonutils.Marshal(map[string]interface{}{
			"storage_id":   storage.Id,
			"storage_conf": storage.StorageConf,
		})
		_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, headers, body, false)
		//这里尽可能的更新所有在线的hoststorage信息,仅打印warning信息
		log.Warningf("update rbd storage info for host %s(%s) error: %v", host.Name, host.Id, err)
	}
	self.SetStageComplete(ctx, nil)
}

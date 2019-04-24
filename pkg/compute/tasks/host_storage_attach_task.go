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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type HostStorageAttachTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(HostStorageAttachTask{})
}

func (self *HostStorageAttachTask) taskFail(ctx context.Context, host *models.SHost, reason string) {
	if hoststorage := self.getHoststorage(host); hoststorage != nil {
		storage := hoststorage.GetStorage()
		hoststorage.Detach(ctx, self.GetUserCred())
		note := fmt.Sprintf("attach host %s failed: %s", host.Name, reason)
		db.OpsLog.LogEvent(storage, db.ACT_ATTACH, note, self.GetUserCred())
	}
	self.SetStageFailed(ctx, reason)
}

func (self *HostStorageAttachTask) getHoststorage(host *models.SHost) *models.SHoststorage {
	storageId, _ := self.GetParams().GetString("storage_id")
	hoststorages := host.GetHoststorages()
	for _, hoststorage := range hoststorages {
		if hoststorage.StorageId == storageId {
			return &hoststorage
		}
	}
	return nil
}

func (self *HostStorageAttachTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	host := obj.(*models.SHost)
	hoststorage := self.getHoststorage(host)
	if hoststorage == nil {
		self.taskFail(ctx, host, "failed to find hoststorage")
		return
	}
	storage := hoststorage.GetStorage()
	self.SetStage("OnAttachStorageComplete", nil)
	err := host.GetHostDriver().RequestAttachStorage(ctx, hoststorage, host, storage, self)
	if err != nil {
		self.taskFail(ctx, host, err.Error())
	}
}

func (self *HostStorageAttachTask) OnAttachStorageComplete(ctx context.Context, host *models.SHost, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *HostStorageAttachTask) OnAttachStorageCompleteFailed(ctx context.Context, host *models.SHost, reason jsonutils.JSONObject) {
	self.taskFail(ctx, host, reason.String())
}

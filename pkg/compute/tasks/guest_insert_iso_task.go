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

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type GuestInsertIsoTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestInsertIsoTask{})
}

func (self *GuestInsertIsoTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.prepareIsoImage(ctx, obj)
}

func (self *GuestInsertIsoTask) prepareIsoImage(ctx context.Context, obj db.IStandaloneModel) {
	guest := obj.(*models.SGuest)
	imageId, _ := self.Params.GetString("image_id")
	db.OpsLog.LogEvent(obj, db.ACT_ISO_PREPARING, imageId, self.UserCred)

	disks := guest.GetDisks()
	disk := disks[0].GetDisk()
	storage := disk.GetStorage()
	storageCache := storage.GetStoragecache()

	if storageCache != nil {
		self.SetStage("OnIsoPrepareComplete", nil)
		storageCache.StartImageCacheTask(ctx, self.UserCred, imageId, "iso", false, self.GetTaskId())
	} else {
		guest.EjectIso(self.UserCred)
		db.OpsLog.LogEvent(obj, db.ACT_ISO_PREPARE_FAIL, imageId, self.UserCred)
		self.SetStageFailed(ctx, "host no local storage cache")
	}
}

func (self *GuestInsertIsoTask) OnIsoPrepareCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	imageId, _ := self.Params.GetString("image_id")
	db.OpsLog.LogEvent(obj, db.ACT_ISO_PREPARE_FAIL, imageId, self.UserCred)
	guest := obj.(*models.SGuest)
	guest.EjectIso(self.UserCred)
	self.SetStageFailed(ctx, "OnIsoPrepareCompleteFailed")
}

func (self *GuestInsertIsoTask) OnIsoPrepareComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	imageId, _ := data.GetString("image_id")
	size, err := data.Int("size")
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
		return
	}
	name, _ := data.GetString("name")
	path, _ := data.GetString("path")
	guest := obj.(*models.SGuest)
	if guest.InsertIsoSucc(imageId, path, int(size), name) {
		db.OpsLog.LogEvent(guest, db.ACT_ISO_ATTACH, guest.GetDetailsIso(self.UserCred), self.UserCred)
		if guest.GetDriver().NeedRequestGuestHotAddIso(ctx, guest) {
			self.SetStage("OnConfigSyncComplete", nil)
			boot := jsonutils.QueryBoolean(self.Params, "boot", false)
			guest.GetDriver().RequestGuestHotAddIso(ctx, guest, path, boot, self)
		} else {
			self.SetStageComplete(ctx, nil)
		}
	} else {
		self.SetStageComplete(ctx, nil)
	}
}

func (self *GuestInsertIsoTask) OnConfigSyncComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

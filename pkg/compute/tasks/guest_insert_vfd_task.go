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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type GuestInsertVfdTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestInsertVfdTask{})
	taskman.RegisterTask(HaGuestInsertVfdTask{})
}

func (self *GuestInsertVfdTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.prepareVfdImage(ctx, obj)
}

func (self *GuestInsertVfdTask) prepareVfdImage(ctx context.Context, obj db.IStandaloneModel) {
	guest := obj.(*models.SGuest)
	imageId, _ := self.Params.GetString("image_id")
	floppyOrdinal, _ := self.Params.Int("floppy_ordinal")
	db.OpsLog.LogEvent(obj, db.ACT_VFD_PREPARING, imageId, self.UserCred)

	disks, _ := guest.GetGuestDisks()
	disk := disks[0].GetDisk()
	storage, _ := disk.GetStorage()
	storageCache := storage.GetStoragecache()

	if storageCache != nil {
		self.SetStage("OnVfdPrepareComplete", nil)
		input := api.CacheImageInput{
			ImageId:      imageId,
			Format:       "raw",
			ParentTaskId: self.GetTaskId(),
		}
		storageCache.StartImageCacheTask(ctx, self.UserCred, input)
	} else {
		guest.EjectVfd(floppyOrdinal, self.UserCred)
		db.OpsLog.LogEvent(obj, db.ACT_VFD_PREPARE_FAIL, imageId, self.UserCred)
		self.SetStageFailed(ctx, jsonutils.NewString("host no local storage cache"))
	}
}

func (self *GuestInsertVfdTask) OnVfdPrepareCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	imageId, _ := self.Params.GetString("image_id")
	floppyOrdinal, _ := self.Params.Int("floppy_ordinal")
	db.OpsLog.LogEvent(obj, db.ACT_VFD_PREPARE_FAIL, imageId, self.UserCred)
	guest := obj.(*models.SGuest)
	guest.EjectVfd(floppyOrdinal, self.UserCred)
	self.SetStageFailed(ctx, data)
}

func (self *GuestInsertVfdTask) OnVfdPrepareComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	floppyOrdinal, _ := self.Params.Int("floppy_ordinal")
	imageId, _ := data.GetString("image_id")
	size, err := data.Int("size")
	if err != nil {
		self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}
	name, _ := data.GetString("name")
	path, _ := data.GetString("path")
	guest := obj.(*models.SGuest)
	if guest.InsertVfdSucc(floppyOrdinal, imageId, path, size, name) {
		db.OpsLog.LogEvent(guest, db.ACT_VFD_ATTACH, guest.GetDetailsVfd(floppyOrdinal, self.UserCred), self.UserCred)
		drv, err := guest.GetDriver()
		if err != nil {
			self.OnVfdPrepareCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
			return
		}
		if drv.NeedRequestGuestHotAddVfd(ctx, guest) {
			self.SetStage("OnConfigSyncComplete", nil)
			boot := jsonutils.QueryBoolean(self.Params, "boot", false)
			drv.RequestGuestHotAddVfd(ctx, guest, path, boot, self)
		} else {
			self.SetStageComplete(ctx, nil)
		}
	} else {
		self.SetStageComplete(ctx, nil)
	}
}

func (self *GuestInsertVfdTask) OnConfigSyncComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

type HaGuestInsertVfdTask struct {
	GuestInsertVfdTask
}

func (self *HaGuestInsertVfdTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.prepareVfdImage(ctx, obj)
}

func (self *HaGuestInsertVfdTask) prepareVfdImage(ctx context.Context, obj db.IStandaloneModel) {
	guest := obj.(*models.SGuest)
	imageId, _ := self.Params.GetString("image_id")
	floppyOrdinal, _ := self.Params.Int("floppy_ordinal")
	db.OpsLog.LogEvent(obj, db.ACT_VFD_PREPARING, imageId, self.UserCred)
	disks, _ := guest.GetGuestDisks()
	disk := disks[0].GetDisk()
	storage := disk.GetBackupStorage()
	storageCache := storage.GetStoragecache()
	if storageCache != nil {
		self.SetStage("OnBackupVfdPrepareComplete", nil)
		input := api.CacheImageInput{
			ImageId:      imageId,
			Format:       "raw",
			ParentTaskId: self.GetTaskId(),
		}
		storageCache.StartImageCacheTask(ctx, self.UserCred, input)
	} else {
		guest.EjectVfd(floppyOrdinal, self.UserCred)
		db.OpsLog.LogEvent(obj, db.ACT_VFD_PREPARE_FAIL, imageId, self.UserCred)
		self.SetStageFailed(ctx, jsonutils.NewString("host no local storage cache"))
	}
}

func (self *HaGuestInsertVfdTask) OnBackupVfdPrepareComplete(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	self.GuestInsertVfdTask.prepareVfdImage(ctx, guest)
}

func (self *HaGuestInsertVfdTask) OnBackupVfdPrepareCompleteFailed(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	self.OnVfdPrepareCompleteFailed(ctx, guest, data)
}

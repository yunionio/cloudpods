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

	api "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/image/models"
	"yunion.io/x/onecloud/pkg/image/options"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestImageDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(GuestImageDeleteTask{})
}

func (self *GuestImageDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guestImage := obj.(*models.SGuestImage)
	isPurge := jsonutils.QueryBoolean(self.Params, "purge", false)
	isOverridePendingDelete := jsonutils.QueryBoolean(self.Params, "override_pending_delete", false)

	if options.Options.EnablePendingDelete && !isPurge && !isOverridePendingDelete {
		if guestImage.PendingDeleted {
			self.SetStageComplete(ctx, nil)
		}
		self.startPendingDelete(ctx, guestImage)
	} else {
		self.startDelete(ctx, guestImage)
	}
}

func (self *GuestImageDeleteTask) startPendingDelete(ctx context.Context, guestImage *models.SGuestImage) {
	images, err := models.GuestImageJointManager.GetImagesByGuestImageId(guestImage.GetId())
	if err != nil {
		self.taskFailed(ctx, guestImage, err.Error())
	}
	for i := range images {
		images[i].StopTorrents()
		err := images[i].DoPendingDelete(ctx, self.UserCred)
		if err != nil {
			self.taskFailed(ctx, guestImage, fmt.Sprintf("image %s pending delete failed", images[i].GetId()))
			return
		}
	}
	err = guestImage.DoPendingDelete(ctx, self.UserCred)
	if err != nil {
		self.taskFailed(ctx, guestImage, fmt.Sprintf("guest image %s pending delete failed", guestImage.GetId()))
	}
	self.SetStageComplete(ctx, nil)
}

func (self *GuestImageDeleteTask) startDelete(ctx context.Context, guestImage *models.SGuestImage) {
	images, err := models.GuestImageJointManager.GetImagesByGuestImageId(guestImage.GetId())
	if err != nil {
		self.taskFailed(ctx, guestImage, err.Error())
	}
	for i := range images {
		err := images[i].RemoveFiles()
		if err != nil {
			self.taskFailed(ctx, guestImage, fmt.Sprintf("fail to remove %s: %s", images[i].GetPath(""), err))
			return
		}
		err = images[i].SetStatus(self.UserCred, api.IMAGE_STATUS_DELETED, "delete")
		if err != nil {
			self.taskFailed(ctx, guestImage, fmt.Sprintf("fail to set image %s status ", images[i].GetId()))
			return
		}
		err = images[i].RealDelete(ctx, self.UserCred)
		if err != nil {
			self.taskFailed(ctx, guestImage, fmt.Sprintf("fail to real delete image %s", images[i].GetId()))
			return
		}
	}
	err = guestImage.SetStatus(self.UserCred, api.IMAGE_STATUS_DELETED, "delete")
	if err != nil {
		self.taskFailed(ctx, guestImage, fmt.Sprintf("fail to set guest image status %s", guestImage.GetId()))
	}
	err = guestImage.RealDelete(ctx, self.UserCred)
	if err != nil {
		self.taskFailed(ctx, guestImage, fmt.Sprintf("fail to real delete guest image %s", guestImage.GetId()))
	}
	self.SetStageComplete(ctx, nil)
}

func (self *GuestImageDeleteTask) taskFailed(ctx context.Context, guestImage *models.SGuestImage, reason string) {
	log.Errorf("Guest Image %s delete failed: %s", guestImage.Id, reason)
	db.OpsLog.LogEvent(guestImage, db.ACT_IMAGE_DELETE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithContext(ctx, guestImage, logclient.ACT_DELETE, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

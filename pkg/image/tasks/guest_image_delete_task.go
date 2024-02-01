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

func (task *GuestImageDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guestImage := obj.(*models.SGuestImage)
	isPurge := jsonutils.QueryBoolean(task.Params, "purge", false)
	isOverridePendingDelete := jsonutils.QueryBoolean(task.Params, "override_pending_delete", false)

	if options.Options.EnablePendingDelete && !isPurge && !isOverridePendingDelete {
		if guestImage.PendingDeleted {
			task.SetStageComplete(ctx, nil)
		} else {
			task.startPendingDelete(ctx, guestImage)
		}
	} else {
		task.startDelete(ctx, guestImage)
	}
}

func (task *GuestImageDeleteTask) startPendingDelete(ctx context.Context, guestImage *models.SGuestImage) {
	images, err := models.GuestImageJointManager.GetImagesByGuestImageId(guestImage.GetId())
	if err != nil {
		task.taskFailed(ctx, guestImage, jsonutils.NewString(err.Error()))
	}
	for i := range images {
		if !images[i].IsGuestImage.IsTrue() {
			continue
		}
		images[i].StopTorrents()
		err := images[i].DoPendingDelete(ctx, task.UserCred)
		if err != nil {
			task.taskFailed(ctx, guestImage, jsonutils.NewString(fmt.Sprintf("image %s pending delete failed", images[i].GetId())))
			return
		}
	}
	err = guestImage.DoPendingDelete(ctx, task.UserCred)
	if err != nil {
		task.taskFailed(ctx, guestImage, jsonutils.NewString(fmt.Sprintf("guest image %s pending delete failed", guestImage.GetId())))
		return
	}
	task.SetStageComplete(ctx, nil)
}

func (task *GuestImageDeleteTask) startDelete(ctx context.Context, guestImage *models.SGuestImage) {
	images, err := models.GuestImageJointManager.GetImagesByGuestImageId(guestImage.GetId())
	if err != nil {
		task.taskFailed(ctx, guestImage, jsonutils.NewString(err.Error()))
		return
	}
	for i := range images {
		if !images[i].IsGuestImage.IsTrue() {
			continue
		}
		err := images[i].Remove(ctx, task.UserCred)
		if err != nil {
			task.taskFailed(ctx, guestImage, jsonutils.NewString(fmt.Sprintf("fail to remove %s: %s", images[i].GetPath(""), err)))
			return
		}
		err = images[i].SetStatus(ctx, task.UserCred, api.IMAGE_STATUS_DELETED, "delete")
		if err != nil {
			task.taskFailed(ctx, guestImage, jsonutils.NewString(fmt.Sprintf("fail to set image %s status ", images[i].GetId())))
			return
		}
		err = images[i].RealDelete(ctx, task.UserCred)
		if err != nil {
			task.taskFailed(ctx, guestImage, jsonutils.NewString(fmt.Sprintf("fail to real delete image %s", images[i].GetId())))
			return
		}
	}
	err = guestImage.SetStatus(ctx, task.UserCred, api.IMAGE_STATUS_DELETED, "delete")
	if err != nil {
		task.taskFailed(ctx, guestImage, jsonutils.NewString(fmt.Sprintf("fail to set guest image status %s", guestImage.GetId())))
		return
	}
	err = guestImage.RealDelete(ctx, task.UserCred)
	if err != nil {
		task.taskFailed(ctx, guestImage, jsonutils.NewString(fmt.Sprintf("fail to real delete guest image %s", guestImage.GetId())))
		return
	}
	task.SetStageComplete(ctx, nil)
}

func (task *GuestImageDeleteTask) taskFailed(ctx context.Context, guestImage *models.SGuestImage, reason jsonutils.JSONObject) {
	log.Errorf("Guest Image %s delete failed: %s", guestImage.Id, reason)
	db.OpsLog.LogEvent(guestImage, db.ACT_IMAGE_DELETE_FAIL, reason, task.UserCred)
	logclient.AddActionLogWithContext(ctx, guestImage, logclient.ACT_DELETE, reason, task.UserCred, false)
	task.SetStageFailed(ctx, reason)
}

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
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/image/models"
	"yunion.io/x/onecloud/pkg/image/options"
)

type ImageDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ImageDeleteTask{})
}

func (self *ImageDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	image := obj.(*models.SImage)

	// imageStatus, _ := self.Params.GetString("image_status")
	isPurge := jsonutils.QueryBoolean(self.Params, "purge", false)
	isOverridePendingDelete := jsonutils.QueryBoolean(self.Params, "override_pending_delete", false)

	if options.Options.EnablePendingDelete && !isPurge && !isOverridePendingDelete {
		// imageStatus == models.IMAGE_STATUS_ACTIVE
		if image.PendingDeleted {
			self.SetStageComplete(ctx, nil)
			return
		}
		self.startPendingDeleteImage(ctx, image)
	} else {
		self.startDeleteImage(ctx, image)
	}
}

func (self *ImageDeleteTask) startPendingDeleteImage(ctx context.Context, image *models.SImage) {
	image.StopTorrents()
	image.DoPendingDelete(ctx, self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *ImageDeleteTask) startDeleteImage(ctx context.Context, image *models.SImage) {
	err := image.Remove(ctx, self.UserCred)
	if err != nil {
		msg := fmt.Sprintf("fail to remove %s %s", image.Name, err)
		log.Errorf(msg)
		self.SetStageFailed(ctx, jsonutils.NewString(msg))
		return
	}

	image.SetStatus(ctx, self.UserCred, api.IMAGE_STATUS_DELETED, "delete")

	image.RealDelete(ctx, self.UserCred)

	self.SetStageComplete(ctx, nil)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    image,
		Action: notifyclient.ActionDelete,
	})
}

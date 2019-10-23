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
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/image/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestImageUpdateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(GuestImageUpdateTask{})
}

func (self *GuestImageUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guestImage := obj.(*models.SGuestImage)
	subImages, err := models.GuestImageJointManager.GetImagesByFilter(guestImage.GetId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Asc("name")
	})
	if err != nil {
		self.taskFailed(ctx, guestImage, err.Error())
	}
	if self.Params.Contains("name") {
		name, _ := self.Params.GetString("name")
		for i := range subImages[:len(subImages)-1] {
			sub := &subImages[i]
			_, err := db.Update(sub, func() error {
				sub.Name = fmt.Sprintf("%s-%s-%d", name, "data", i)
				return nil
			})
			if err != nil {
				self.taskFailed(ctx, guestImage, fmt.Sprintf("modify subimage's name failed: %s", err.Error()))
				return
			}
		}
		root := &subImages[len(subImages)-1]
		_, err := db.Update(root, func() error {
			root.Name = fmt.Sprintf("%s-%s", name, "root")
			return nil
		})
		if err != nil {
			self.taskFailed(ctx, guestImage, fmt.Sprintf("modify subimage's name failed: %s", err.Error()))
			return
		}
	}
	if self.Params.Contains("properties") {
		rootImageId := subImages[len(subImages)-1].GetId()
		props, _ := self.Params.Get("properties")
		err := models.ImagePropertyManager.SaveProperties(ctx, self.UserCred, rootImageId, props)
		if err != nil {
			self.taskFailed(ctx, guestImage, fmt.Sprintf("save properties error: %s", err.Error()))
			return
		}
	}
	self.SetStageComplete(ctx, nil)
}

func (self *GuestImageUpdateTask) taskFailed(ctx context.Context, guestImage *models.SGuestImage, reason string) {
	db.OpsLog.LogEvent(guestImage, db.ACT_SUBIMAGE_UPDATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithContext(ctx, guestImage, logclient.ACT_SUBIMAGE_UPDATE, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

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
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type GuestRemoteUpdateTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestRemoteUpdateTask{})
}

func (self *GuestRemoteUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.SetStage("OnRemoteUpdateComplete", nil)
	replaceTags := jsonutils.QueryBoolean(self.Params, "replace_tags", false)
	taskman.LocalTaskRun(self, func() (jsonutils.JSONObject, error) {
		drv, err := guest.GetDriver()
		if err != nil {
			return nil, err
		}
		err = drv.RequestRemoteUpdate(ctx, guest, self.UserCred, replaceTags)
		if err != nil {
			return nil, errors.Wrap(err, "RequestRemoteUpdate")
		}
		return nil, nil
	})

}

func (self *GuestRemoteUpdateTask) OnRemoteUpdateComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStage("OnSyncStatusComplete", nil)
	guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
}

func (self *GuestRemoteUpdateTask) OnRemoteUpdateCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetStatus(ctx, self.UserCred, api.VM_UPDATE_TAGS_FAILED, data.String())
	self.SetStageFailed(ctx, data)
}

func (self *GuestRemoteUpdateTask) OnSyncStatusComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestRemoteUpdateTask) OnSyncStatusCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}

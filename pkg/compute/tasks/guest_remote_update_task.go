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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestRemoteUpdateTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestRemoteUpdateTask{})
}

func (self *GuestRemoteUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_SYNC_CONF, nil, self.UserCred)
	self.SetStage("OnRemoteUpdateComplete", nil)
	replaceTags := jsonutils.QueryBoolean(self.Params, "replace_tags", false)
	taskman.LocalTaskRun(self, func() (jsonutils.JSONObject, error) {
		if err := guest.GetDriver().RequestRemoteUpdate(ctx, guest, self.UserCred, replaceTags); err != nil {
			logclient.AddActionLogWithStartable(self, guest, logclient.ACT_UPDATE, err, self.UserCred, false)
			log.Errorf("RequestRemoteUpdate faled %v", err)
			return nil, errors.Wrap(err, "RequestRemoteUpdate")
		}
		return nil, nil
	})

}

func (self *GuestRemoteUpdateTask) OnRemoteUpdateComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	if !self.IsSubtask() {
		self.SetStage("OnSyncStatusComplete", nil)
		guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
	} else {
		self.OnSyncStatusComplete(ctx, obj, data)
	}
}

func (self *GuestRemoteUpdateTask) OnSyncStatusComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	// guest := obj.(*models.SGuest)
	self.SetStageComplete(ctx, nil)
}

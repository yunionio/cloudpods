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
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ElasticSearchSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ElasticSearchSyncstatusTask{})
}

func (self *ElasticSearchSyncstatusTask) taskFailed(ctx context.Context, es *models.SElasticSearch, err error) {
	es.SetStatus(ctx, self.UserCred, api.ELASTIC_SEARCH_STATUS_UNKNOWN, err.Error())
	db.OpsLog.LogEvent(es, db.ACT_SYNC_STATUS, err, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, es, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *ElasticSearchSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	es := obj.(*models.SElasticSearch)

	iEs, err := es.GetIElasticSearch(ctx)
	if err != nil {
		self.taskFailed(ctx, es, errors.Wrapf(err, "es.GetIElasticSearch"))
		return
	}
	es.SyncWithCloudElasticSearch(ctx, self.UserCred, iEs)
	self.SetStageComplete(ctx, nil)
}

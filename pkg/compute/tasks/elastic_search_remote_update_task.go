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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type ElasticSearchRemoteUpdateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ElasticSearchRemoteUpdateTask{})
}

func (self *ElasticSearchRemoteUpdateTask) taskFail(ctx context.Context, elasticcache *models.SElasticSearch, reason jsonutils.JSONObject) {
	elasticcache.SetStatus(ctx, self.UserCred, api.ELASTIC_SEARCH_UPDATE_TAGS_FAILED, reason.String())
	self.SetStageFailed(ctx, reason)
}

func (self *ElasticSearchRemoteUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	es := obj.(*models.SElasticSearch)
	region, _ := es.GetRegion()
	if region == nil {
		self.taskFail(ctx, es, jsonutils.NewString(fmt.Sprintf("failed to find region for elastic search %s", es.GetName())))
		return
	}
	self.SetStage("OnRemoteUpdateComplete", nil)
	replaceTags := jsonutils.QueryBoolean(self.Params, "replace_tags", false)

	if err := region.GetDriver().RequestRemoteUpdateElasticSearch(ctx, self.GetUserCred(), es, replaceTags, self); err != nil {
		self.taskFail(ctx, es, jsonutils.NewString(err.Error()))
	}
}

func (self *ElasticSearchRemoteUpdateTask) OnRemoteUpdateComplete(ctx context.Context, elasticcache *models.SElasticSearch, data jsonutils.JSONObject) {
	self.SetStage("OnSyncStatusComplete", nil)
	elasticcache.StartSElasticSearchSyncTask(ctx, self.UserCred, self.GetTaskId())
}

func (self *ElasticSearchRemoteUpdateTask) OnRemoteUpdateCompleteFailed(ctx context.Context, elasticcache *models.SElasticSearch, data jsonutils.JSONObject) {
	self.taskFail(ctx, elasticcache, data)
}

func (self *ElasticSearchRemoteUpdateTask) OnSyncStatusComplete(ctx context.Context, elasticcache *models.SElasticSearch, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *ElasticSearchRemoteUpdateTask) OnSyncStatusCompleteFailed(ctx context.Context, elasticcache *models.SElasticSearch, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}

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

type KafkaRemoteUpdateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(KafkaRemoteUpdateTask{})
}

func (self *KafkaRemoteUpdateTask) taskFail(ctx context.Context, kafka *models.SKafka, reason jsonutils.JSONObject) {
	kafka.SetStatus(ctx, self.UserCred, api.KAFKA_UPDATE_TAGS_FAILED, reason.String())
	self.SetStageFailed(ctx, reason)
}

func (self *KafkaRemoteUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	kafka := obj.(*models.SKafka)
	region, _ := kafka.GetRegion()
	if region == nil {
		self.taskFail(ctx, kafka, jsonutils.NewString(fmt.Sprintf("failed to find region for elastic search %s", kafka.GetName())))
		return
	}
	self.SetStage("OnRemoteUpdateComplete", nil)
	replaceTags := jsonutils.QueryBoolean(self.Params, "replace_tags", false)

	if err := region.GetDriver().RequestRemoteUpdateKafka(ctx, self.GetUserCred(), kafka, replaceTags, self); err != nil {
		self.taskFail(ctx, kafka, jsonutils.NewString(err.Error()))
	}
}

func (self *KafkaRemoteUpdateTask) OnRemoteUpdateComplete(ctx context.Context, kafka *models.SKafka, data jsonutils.JSONObject) {
	self.SetStage("OnSyncStatusComplete", nil)
	kafka.StartKafkaSyncTask(ctx, self.UserCred, self.GetTaskId())
}

func (self *KafkaRemoteUpdateTask) OnRemoteUpdateCompleteFailed(ctx context.Context, kafka *models.SKafka, data jsonutils.JSONObject) {
	self.taskFail(ctx, kafka, data)
}

func (self *KafkaRemoteUpdateTask) OnSyncStatusComplete(ctx context.Context, kafka *models.SKafka, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *KafkaRemoteUpdateTask) OnSyncStatusCompleteFailed(ctx context.Context, kafka *models.SKafka, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}

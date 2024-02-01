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

type KafkaSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(KafkaSyncstatusTask{})
}

func (self *KafkaSyncstatusTask) taskFailed(ctx context.Context, kafka *models.SKafka, err error) {
	kafka.SetStatus(ctx, self.UserCred, api.KAFKA_STATUS_UNKNOWN, err.Error())
	db.OpsLog.LogEvent(kafka, db.ACT_SYNC_STATUS, err, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, kafka, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *KafkaSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	kafka := obj.(*models.SKafka)

	iKafka, err := kafka.GetIKafka(ctx)
	if err != nil {
		self.taskFailed(ctx, kafka, errors.Wrapf(err, "kafka.GetIKafka"))
		return
	}
	kafka.SyncWithCloudKafka(ctx, self.UserCred, iKafka)
	self.SetStageComplete(ctx, nil)
}

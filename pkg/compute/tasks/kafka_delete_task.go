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
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type KafkaDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(KafkaDeleteTask{})
}

func (self *KafkaDeleteTask) taskFail(ctx context.Context, kafka *models.SKafka, err error) {
	kafka.SetStatus(ctx, self.GetUserCred(), api.KAFKA_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(kafka, db.ACT_DELOCATE_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, kafka, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *KafkaDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	kafka := obj.(*models.SKafka)

	iKafka, err := kafka.GetIKafka(ctx)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, kafka)
			return
		}
		self.taskFail(ctx, kafka, errors.Wrapf(err, "GetIKafka"))
		return
	}
	err = iKafka.Delete()
	if err != nil {
		self.taskFail(ctx, kafka, errors.Wrapf(err, "iKafka.Delete"))
		return
	}
	cloudprovider.WaitDeleted(iKafka, time.Second*10, time.Minute*5)
	self.taskComplete(ctx, kafka)
}

func (self *KafkaDeleteTask) taskComplete(ctx context.Context, kafka *models.SKafka) {
	kafka.RealDelete(ctx, self.GetUserCred())
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    self,
		Action: notifyclient.ActionDelete,
	})
	self.SetStageComplete(ctx, nil)
}

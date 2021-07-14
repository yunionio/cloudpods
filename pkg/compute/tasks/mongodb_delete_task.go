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

// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or fsreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific langufse governing permissions and
// limitations under the License.

package tasks

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type MongoDBDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(MongoDBDeleteTask{})
}

func (self *MongoDBDeleteTask) taskFailed(ctx context.Context, mongodb *models.SMongoDB, err error) {
	mongodb.SetStatus(self.UserCred, api.MONGO_DB_STATUS_DELETE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, mongodb, logclient.ACT_DELOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *MongoDBDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	mongodb := obj.(*models.SMongoDB)

	iMongoDB, err := mongodb.GetIMongoDB()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, mongodb)
			return
		}
		self.taskFailed(ctx, mongodb, errors.Wrapf(err, "mongodb.GetIMongoDB"))
		return
	}
	err = iMongoDB.Delete()
	if err != nil {
		self.taskFailed(ctx, mongodb, errors.Wrapf(err, "iMongoDB.Delete"))
		return
	}
	cloudprovider.WaitDeleted(iMongoDB, time.Second*10, time.Minute*5)
	self.taskComplete(ctx, mongodb)
}

func (self *MongoDBDeleteTask) taskComplete(ctx context.Context, mongodb *models.SMongoDB) {
	mongodb.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

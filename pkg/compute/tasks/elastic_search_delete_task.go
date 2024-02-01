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

type ElasticSearchDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ElasticSearchDeleteTask{})
}

func (self *ElasticSearchDeleteTask) taskFail(ctx context.Context, es *models.SElasticSearch, err error) {
	es.SetStatus(ctx, self.GetUserCred(), api.ELASTIC_SEARCH_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(es, db.ACT_DELOCATE_FAIL, err, self.UserCred)
	// 记录删除失败日志
	logclient.AddActionLogWithStartable(self, es, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *ElasticSearchDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	es := obj.(*models.SElasticSearch)

	iEs, err := es.GetIElasticSearch(ctx)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, es)
			return
		}
		self.taskFail(ctx, es, errors.Wrapf(err, "GetIElasticSearch"))
		return
	}
	err = iEs.Delete()
	if err != nil {
		self.taskFail(ctx, es, errors.Wrapf(err, "iEs.Delete"))
		return
	}
	cloudprovider.WaitDeleted(iEs, time.Second*10, time.Minute*5)
	self.taskComplete(ctx, es)
}

func (self *ElasticSearchDeleteTask) taskComplete(ctx context.Context, es *models.SElasticSearch) {
	es.RealDelete(ctx, self.GetUserCred())
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    es,
		Action: notifyclient.ActionDelete,
	})
	self.SetStageComplete(ctx, nil)
}

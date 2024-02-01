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

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type KubeClusterDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(KubeClusterDeleteTask{})
}

func (self *KubeClusterDeleteTask) taskFailed(ctx context.Context, cluster *models.SKubeCluster, err error) {
	cluster.SetStatus(ctx, self.UserCred, apis.STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(cluster, db.ACT_DELOCATE_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, cluster, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *KubeClusterDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	cluster := obj.(*models.SKubeCluster)

	iCluster, err := cluster.GetIKubeCluster(ctx)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, cluster)
			return
		}
		self.taskFailed(ctx, cluster, errors.Wrapf(err, "GetIKubeCluster"))
		return
	}

	isRetain := jsonutils.QueryBoolean(self.GetParams(), "retain", false)
	err = iCluster.Delete(isRetain)
	if err != nil {
		self.taskFailed(ctx, cluster, errors.Wrapf(err, "Delete"))
		return
	}

	err = cloudprovider.WaitDeleted(iCluster, time.Second*10, time.Minute*10)
	if err != nil {
		self.taskFailed(ctx, cluster, errors.Wrapf(err, "WaitDeleted"))
		return
	}
	self.taskComplete(ctx, cluster)
}

func (self *KubeClusterDeleteTask) taskComplete(ctx context.Context, cluster *models.SKubeCluster) {
	logclient.AddActionLogWithStartable(self, cluster, logclient.ACT_DELETE, cluster.GetShortDesc(ctx), self.UserCred, true)
	cluster.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

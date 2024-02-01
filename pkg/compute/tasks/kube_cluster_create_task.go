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

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type KubeClusterCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(KubeClusterCreateTask{})
}

func (self *KubeClusterCreateTask) taskFail(ctx context.Context, cluster *models.SKubeCluster, err error) {
	cluster.SetStatus(ctx, self.UserCred, apis.STATUS_CREATE_FAILED, err.Error())
	db.OpsLog.LogEvent(cluster, db.ACT_ALLOCATE, err, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, cluster, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *KubeClusterCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*models.SKubeCluster)

	region, err := cluster.GetRegion()
	if err != nil {
		self.taskFail(ctx, cluster, errors.Wrapf(err, "GetRegion"))
		return
	}

	self.SetStage("OnKubeClusterCreateComplate", nil)
	err = region.GetDriver().RequestCreateKubeCluster(ctx, self.UserCred, cluster, self)
	if err != nil {
		self.taskFail(ctx, cluster, errors.Wrapf(err, "RequestCreateKubeCluster"))
		return
	}
}

func (self *KubeClusterCreateTask) OnKubeClusterCreateComplate(ctx context.Context, cluster *models.SKubeCluster, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *KubeClusterCreateTask) OnKubeClusterCreateComplateFailed(ctx context.Context, cluster *models.SKubeCluster, reason jsonutils.JSONObject) {
	self.taskFail(ctx, cluster, errors.Errorf(reason.String()))
}

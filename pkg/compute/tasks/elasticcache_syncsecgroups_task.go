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
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ElasticcacheSyncsecgroupsTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ElasticcacheSyncsecgroupsTask{})
}

func (self *ElasticcacheSyncsecgroupsTask) taskFailed(ctx context.Context, cache *models.SElasticcache, err error) {
	cache.SetStatus(ctx, self.GetUserCred(), api.ELASTIC_CACHE_STATUS_SYNC_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
	db.OpsLog.LogEvent(cache, db.ACT_SYNC_CONF, cache.GetShortDesc(ctx), self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, cache, logclient.ACT_SYNC_CONF, err, self.UserCred, false)
}

func (self *ElasticcacheSyncsecgroupsTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cache := obj.(*models.SElasticcache)

	region, err := cache.GetRegion()
	if err != nil {
		self.taskFailed(ctx, cache, errors.Wrapf(err, "GetRegion"))
		return
	}

	self.SetStage("OnElasticcacheSyncSecgroupsComplete", nil)
	err = region.GetDriver().RequestSyncSecgroupsForElasticcache(ctx, self.GetUserCred(), cache, self)
	if err != nil {
		self.taskFailed(ctx, cache, errors.Wrapf(err, "RequestSyncSecgroupsForElasticcache"))
		return
	}
}

// https://cloud.tencent.com/document/api/239/41256
func (self *ElasticcacheSyncsecgroupsTask) OnElasticcacheSyncSecgroupsComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cache := obj.(*models.SElasticcache)
	secgroups := []string{}
	err := data.Unmarshal(&secgroups, "ext_secgroup_ids")
	if err != nil {
		self.taskFailed(ctx, cache, errors.Wrapf(err, "get ext_secgroup_ids"))
		return
	}

	iregion, err := cache.GetIRegion(ctx)
	if err != nil {
		self.taskFailed(ctx, cache, errors.Wrapf(err, "GetIRegion"))
		return
	}

	iec, err := iregion.GetIElasticcacheById(cache.GetExternalId())
	if err != nil {
		self.taskFailed(ctx, cache, errors.Wrapf(err, "GetIElasticcacheById"))
		return
	}

	err = iec.UpdateSecurityGroups(secgroups)
	if err != nil {
		self.taskFailed(ctx, cache, errors.Wrapf(err, "UpdateSecurityGroups"))
		return
	}

	cache.SetStatus(ctx, self.UserCred, iec.GetStatus(), "UpdateSecurityGroups")
	self.SetStageComplete(ctx, nil)
}

func (self *ElasticcacheSyncsecgroupsTask) OnElasticcacheSyncSecgroupsCompleteFailed(ctx context.Context, cache *models.SElasticcache, data jsonutils.JSONObject) {
	self.taskFailed(ctx, cache, fmt.Errorf(data.String()))
}

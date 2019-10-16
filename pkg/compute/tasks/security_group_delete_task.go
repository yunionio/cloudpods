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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SecurityGroupDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SecurityGroupDeleteTask{})
}

func (self *SecurityGroupDeleteTask) getErrorCount() int64 {
	count, _ := self.GetParams().Int("faild_count")
	return count
}

func (self *SecurityGroupDeleteTask) addErrorCount() {
	count := self.getErrorCount()
	count += 1
	self.GetParams().Set("failed_count", jsonutils.NewInt(count))
}

func (self *SecurityGroupDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStage("OnSecurityGroupUncacheComplete", nil)
	self.OnSecurityGroupUncacheComplete(ctx, obj, data)
}

func (self *SecurityGroupDeleteTask) OnSecurityGroupUncacheComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	secgroup := obj.(*models.SSecurityGroup)
	secgroupCaches := secgroup.GetSecurityGroupCaches()
	errCount := self.getErrorCount()
	if len(secgroupCaches) == int(errCount) {
		if errCount == 0 {
			secgroup.RealDelete(ctx, self.UserCred)
		}
		secgroup.SetStatus(self.UserCred, api.SECGROUP_STATUS_READY, "")
		self.SetStageComplete(ctx, nil)
		return
	}
	secgroupCaches[errCount].StartSecurityGroupCacheDeleteTask(ctx, self.UserCred, self.GetTaskId())
}

func (self *SecurityGroupDeleteTask) OnSecurityGroupUncacheCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	self.addErrorCount()
}

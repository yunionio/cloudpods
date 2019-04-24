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

func (self *SecurityGroupDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	secgroup := obj.(*models.SSecurityGroup)
	secgroupCache := secgroup.GetSecurityGroupCaches()
	for _, cache := range secgroupCache {
		cache.Delete(ctx, self.GetUserCred())
	}
	secgroup.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

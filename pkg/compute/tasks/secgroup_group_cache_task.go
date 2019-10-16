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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SecurityGroupCacheTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SecurityGroupCacheTask{})
}

func (self *SecurityGroupCacheTask) taskFailed(ctx context.Context, secgroup *models.SSecurityGroup, err error) {
	self.SetStageFailed(ctx, err.Error())
}

func (self *SecurityGroupCacheTask) getVpc() (*models.SVpc, error) {
	vpcId, _ := self.GetParams().GetString("vpc_id")
	if len(vpcId) == 0 {
		return nil, fmt.Errorf("Missing vpc_id params")
	}
	vpc, err := models.VpcManager.FetchById(vpcId)
	if err != nil {
		return nil, errors.Wrap(err, "VpcManager.FetchById")
	}
	return vpc.(*models.SVpc), nil
}

func (self *SecurityGroupCacheTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	secgroup := obj.(*models.SSecurityGroup)

	vpc, err := self.getVpc()
	if err != nil {
		self.taskFailed(ctx, secgroup, errors.Wrap(err, "self.getVpc()"))
		return
	}

	region, err := vpc.GetRegion()
	if err != nil {
		self.taskFailed(ctx, secgroup, errors.Wrap(err, "vpc.GetRegion"))
		return
	}

	classic, _ := self.GetParams().Bool("classic")

	self.SetStage("OnCacheSecurityGroupComplete", nil)

	err = region.GetDriver().RequestCacheSecurityGroup(ctx, self.UserCred, region, vpc, secgroup, classic, self)
	if err != nil {
		self.taskFailed(ctx, secgroup, errors.Wrap(err, "RequestCacheSecgroup"))
		return
	}
}

func (self *SecurityGroupCacheTask) OnCacheSecurityGroupComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *SecurityGroupCacheTask) OnCacheSecurityGroupCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	self.SetStageFailed(ctx, err.String())
}

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

type SecurityGroupDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SecurityGroupDeleteTask{})
}

func (self *SecurityGroupDeleteTask) taskFailed(ctx context.Context, secgroup *models.SSecurityGroup, err error) {
	secgroup.SetStatus(self.UserCred, api.SECGROUP_STATUS_READY, "")
	logclient.AddActionLogWithContext(ctx, secgroup, logclient.ACT_DELOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *SecurityGroupDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	secgroup := obj.(*models.SSecurityGroup)
	caches, err := secgroup.GetSecurityGroupCaches()
	if err != nil {
		self.taskFailed(ctx, secgroup, errors.Wrapf(err, "GetSecurityGroupCaches"))
		return
	}

	isPurge := jsonutils.QueryBoolean(self.Params, "purge", false)

	for i := range caches {
		if !isPurge {
			iSecgroup, err := caches[i].GetISecurityGroup(ctx)
			if err != nil {
				if errors.Cause(err) == cloudprovider.ErrNotFound {
					caches[i].RealDelete(ctx, self.GetUserCred())
					continue
				}
				self.taskFailed(ctx, secgroup, errors.Wrapf(err, "GetISecurityGroup for cache %s(%s)", caches[i].Name, caches[i].Id))
				return
			}
			err = iSecgroup.Delete()
			if err != nil {
				self.taskFailed(ctx, secgroup, errors.Wrapf(err, "iSecgroup.Delete"))
				return
			}
		}
		caches[i].RealDelete(ctx, self.GetUserCred())
	}

	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    secgroup,
		Action: notifyclient.ActionDelete,
	})
	secgroup.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

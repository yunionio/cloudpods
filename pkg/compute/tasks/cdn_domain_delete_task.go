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
	"database/sql"

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

type CDNDomainDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CDNDomainDeleteTask{})
}

func (self *CDNDomainDeleteTask) taskFailed(ctx context.Context, domain *models.SCDNDomain, err error) {
	domain.SetStatus(ctx, self.UserCred, api.CDN_DOMAIN_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(domain, db.ACT_DELOCATE_FAIL, err.Error(), self.UserCred)
	logclient.AddActionLogWithStartable(self, domain, logclient.ACT_DELETE, err.Error(), self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *CDNDomainDeleteTask) taskComplete(ctx context.Context, domain *models.SCDNDomain) {
	domain.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

func (self *CDNDomainDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	domain := obj.(*models.SCDNDomain)

	iDomain, err := domain.GetICloudCDNDomain(ctx)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound || errors.Cause(err) == sql.ErrNoRows {
			self.taskComplete(ctx, domain)
			return
		}
		self.taskFailed(ctx, domain, errors.Wrapf(err, "GetICloudCDNDomain"))
		return
	}

	err = iDomain.Delete()
	if err != nil {
		self.taskFailed(ctx, domain, errors.Wrapf(err, "Delete"))
		return
	}

	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    domain,
		Action: notifyclient.ActionDelete,
	})

	self.taskComplete(ctx, domain)
}

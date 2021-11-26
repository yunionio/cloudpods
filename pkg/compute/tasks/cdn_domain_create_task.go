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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type CDNDomainCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CDNDomainCreateTask{})
}

func (self *CDNDomainCreateTask) taskFailed(ctx context.Context, domain *models.SCDNDomain, err error) {
	domain.SetStatus(self.UserCred, apis.STATUS_CREATE_FAILED, err.Error())
	db.OpsLog.LogEvent(domain, db.ACT_DELOCATE_FAIL, err.Error(), self.UserCred)
	logclient.AddActionLogWithStartable(self, domain, logclient.ACT_DELETE, err.Error(), self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *CDNDomainCreateTask) taskComplete(ctx context.Context, domain *models.SCDNDomain) {
	self.SetStageComplete(ctx, nil)
}

func (self *CDNDomainCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	domain := obj.(*models.SCDNDomain)

	driver, err := domain.GetDriver()
	if err != nil {
		self.taskFailed(ctx, domain, errors.Wrapf(err, "GetDriver"))
		return
	}

	opts := cloudprovider.CdnCreateOptions{
		Domain:      domain.Name,
		ServiceType: domain.ServiceType,
		Area:        domain.Area,
		Origins:     cloudprovider.SCdnOrigins{},
	}
	if domain.Origins != nil {
		opts.Origins = *domain.Origins
	}

	iCdn, err := driver.CreateICloudCDNDomain(&opts)
	if err != nil {
		self.taskFailed(ctx, domain, errors.Wrapf(err, "CreateICloudCDNDomain"))
		return
	}

	cloudprovider.WaitMultiStatus(iCdn, []string{
		api.CDN_DOMAIN_STATUS_ONLINE,
		api.CDN_DOMAIN_STATUS_OFFLINE,
		api.CDN_DOMAIN_STATUS_REJECTED,
		api.CDN_DOMAIN_STATUS_UNKNOWN,
	}, time.Second*5, time.Minute*3)

	domain.SyncWithCloudCDNDomain(ctx, self.GetUserCred(), iCdn)

	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    domain,
		Action: notifyclient.ActionCreate,
	})

	self.taskComplete(ctx, domain)
}

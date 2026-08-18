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

package ipset

import (
	"context"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type IpSetCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(IpSetCreateTask{})
}

func (self *IpSetCreateTask) taskFailed(ctx context.Context, ipset *models.SIpSet, err error) {
	ipset.SetStatus(ctx, self.UserCred, apis.STATUS_CREATE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, ipset, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *IpSetCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	ipset := obj.(*models.SIpSet)
	opts := &cloudprovider.IpSetCreateOptions{
		Name:      ipset.Name,
		Desc:      ipset.Description,
		IpSetType: string(ipset.IpSetType),
		Addresses: ipset.GetAddresses(),
	}
	var iIpSet cloudprovider.ICloudIpSet
	var err error
	if ipset.IsManaged() && len(ipset.CloudregionId) == 0 {
		provider := ipset.GetCloudprovider()
		if provider == nil {
			self.taskFailed(ctx, ipset, errors.Errorf("empty cloudprovider"))
			return
		}
		driver, err := provider.GetProvider(ctx)
		if err != nil {
			self.taskFailed(ctx, ipset, errors.Wrapf(err, "GetProvider"))
			return
		}
		iIpSet, err = driver.CreateIIpSet(opts)
	} else {
		iRegion, err := ipset.GetIRegion(ctx)
		if err != nil {
			self.taskFailed(ctx, ipset, errors.Wrapf(err, "GetIRegion"))
			return
		}
		iIpSet, err = iRegion.CreateIIpSet(opts)
	}
	if err != nil {
		self.taskFailed(ctx, ipset, errors.Wrapf(err, "CreateIIpSet"))
		return
	}
	_, err = db.Update(ipset, func() error {
		ipset.ExternalId = iIpSet.GetGlobalId()
		ipset.Status = apis.STATUS_AVAILABLE
		return nil
	})
	if err != nil {
		self.taskFailed(ctx, ipset, errors.Wrapf(err, "db.Update"))
		return
	}
	logclient.AddActionLogWithStartable(self, ipset, logclient.ACT_ALLOCATE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

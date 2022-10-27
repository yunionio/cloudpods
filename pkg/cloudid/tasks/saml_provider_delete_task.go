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

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudid/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SAMLProviderDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SAMLProviderDeleteTask{})
}

func (self *SAMLProviderDeleteTask) taskFailed(ctx context.Context, saml *models.SSAMLProvider, err error) {
	saml.SetStatus(self.GetUserCred(), api.SAML_PROVIDER_STATUS_DELETE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, saml, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *SAMLProviderDeleteTask) taskComplete(ctx context.Context, saml *models.SSAMLProvider) {
	saml.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

func (self *SAMLProviderDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	saml := obj.(*models.SSAMLProvider)

	if len(saml.ExternalId) == 0 {
		self.taskComplete(ctx, saml)
		return
	}

	account, err := saml.GetCloudaccount()
	if err != nil {
		self.taskFailed(ctx, saml, errors.Wrapf(err, "GetCloudaccount"))
		return
	}

	provider, err := account.GetProvider()
	if err != nil {
		self.taskFailed(ctx, saml, errors.Wrapf(err, "GetProvider"))
		return
	}
	samls, err := provider.GetICloudSAMLProviders()
	if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented {
		self.taskFailed(ctx, saml, errors.Wrapf(err, "GetICloudSAMLProviders"))
		return
	}

	for i := range samls {
		if samls[i].GetGlobalId() == saml.ExternalId {
			err = samls[i].Delete()
			if err != nil {
				self.taskFailed(ctx, saml, errors.Wrapf(err, "Delete"))
				return
			}
		}
	}

	self.taskComplete(ctx, saml)
}

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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudid/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SAMLProviderUpdateMetadataTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SAMLProviderUpdateMetadataTask{})
}

func (self *SAMLProviderUpdateMetadataTask) taskFailed(ctx context.Context, saml *models.SSAMLProvider, err error) {
	saml.SetStatus(self.GetUserCred(), api.SAML_PROVIDER_STATUS_UPDATE_METADATA_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, saml, logclient.ACT_UPDATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *SAMLProviderUpdateMetadataTask) taskComplete(ctx context.Context, saml *models.SSAMLProvider) {
	self.SetStageComplete(ctx, nil)
}

func (self *SAMLProviderUpdateMetadataTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	saml := obj.(*models.SSAMLProvider)

	account, err := saml.GetCloudaccount()
	if err != nil {
		self.taskFailed(ctx, saml, errors.Wrapf(err, "GetCloudaccount"))
		return
	}

	metadata := models.SamlIdpInstance().GetMetadata(account.Id)

	provider, err := account.GetProvider()
	if err != nil {
		self.taskFailed(ctx, saml, errors.Wrap(err, "GetProvider"))
		return
	}

	samls, err := provider.GetICloudSAMLProviders()
	if err != nil {
		self.taskFailed(ctx, saml, errors.Wrapf(err, "GetICloudSAMLProviders"))
		return
	}

	var iSaml cloudprovider.ICloudSAMLProvider = nil

	for i := range samls {
		if samls[i].GetGlobalId() == saml.ExternalId {
			iSaml = samls[i]
			break
		}
	}

	if iSaml == nil {
		self.taskFailed(ctx, saml, fmt.Errorf("failed to find saml provider %s(%s)", saml.Name, saml.ExternalId))
		return
	}

	err = iSaml.UpdateMetadata(metadata)
	if err != nil {
		self.taskFailed(ctx, saml, errors.Wrapf(err, "UpdateMetadata"))
		return
	}

	self.SetStage("OnSamlProviderSyncComplete", nil)
	saml.StartSAMLProviderSyncTask(ctx, self.GetUserCred(), self.GetTaskId())
}

func (self *SAMLProviderUpdateMetadataTask) OnSamlProviderSyncComplete(ctx context.Context, saml *models.SSAMLProvider, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *SAMLProviderUpdateMetadataTask) OnSamlProviderSyncCompleteFailed(ctx context.Context, saml *models.SSAMLProvider, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}

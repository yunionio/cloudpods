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

package models

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudid/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/samlutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SSAMLProviderManager struct {
	db.SStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SCloudaccountResourceBaseManager
}

var SAMLProviderManager *SSAMLProviderManager

func init() {
	SAMLProviderManager = &SSAMLProviderManager{
		SStatusInfrasResourceBaseManager: db.NewStatusInfrasResourceBaseManager(
			SSAMLProvider{},
			"saml_provider_tbl",
			"saml_provider",
			"saml_providers",
		),
	}
	SAMLProviderManager.SetVirtualObject(SAMLProviderManager)
}

type SSAMLProvider struct {
	db.SStatusInfrasResourceBase
	db.SExternalizedResourceBase
	SCloudaccountResourceBase

	EntityId         string `get:"domain" create:"domain_optional" list:"domain"`
	MetadataDocument string `get:"domain" create:"domain_optional"`
	AuthUrl          string `width:"512" charset:"ascii" get:"domain" list:"domain"`
}

func (manager *SSAMLProviderManager) GetIVirtualModelManager() db.IVirtualModelManager {
	return manager.GetVirtualObject().(db.IVirtualModelManager)
}

func (manager *SSAMLProviderManager) GetResourceCount() ([]db.SScopeResourceCount, error) {
	return nil, nil
}

func (manager *SSAMLProviderManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	accountId, _ := data.GetString("cloudaccount_id")
	return jsonutils.Marshal(map[string]string{"cloudaccount_id": accountId})
}

func (manager *SSAMLProviderManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	accountId, _ := values.GetString("cloudaccount_id")
	if len(accountId) > 0 {
		q = q.Equals("cloudaccount_id", accountId)
	}
	return q
}

// 创建云账号的身份提供商
func (manager *SSAMLProviderManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.SAMLProviderCreateInput) (api.SAMLProviderCreateInput, error) {
	if len(input.CloudaccountId) == 0 {
		return input, httperrors.NewMissingParameterError("cloudaccount_id")
	}
	_, err := validators.ValidateModel(userCred, CloudaccountManager, &input.CloudaccountId)
	if err != nil {
		return input, err
	}
	input.EntityId = options.Options.ApiServer
	if len(input.EntityId) == 0 {
		return input, httperrors.NewResourceNotReadyError("not set api_server")
	}
	input.Name = strings.TrimPrefix(input.EntityId, "https://")
	input.Name = strings.TrimPrefix(input.Name, "http://")

	input.MetadataDocument = SamlIdpInstance().GetMetadata(input.CloudaccountId).String()
	input.StatusInfrasResourceBaseCreateInput, err = manager.SStatusInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusInfrasResourceBaseCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (self *SSAMLProvider) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.StartSAMLProviderCreateTask(ctx, userCred, "")
}

// 公有云身份提供商列表
func (manager *SSAMLProviderManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.SAMLProviderListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, err
	}
	q, err = manager.SCloudaccountResourceBaseManager.ListItemFilter(ctx, q, userCred, query.CloudaccountResourceListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

// 删除
func (self *SSAMLProvider) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	params := jsonutils.NewDict()
	return self.StartSAMLProviderDeleteTask(ctx, userCred, params, "")
}

func (self *SSAMLProvider) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SSAMLProvider) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SSAMLProvider) StartSAMLProviderDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "SAMLProviderDeleteTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.SAML_PROVIDER_STATUS_DELETING, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SSAMLProvider) IsNeedUpldateMetadata() bool {
	if len(self.ExternalId) == 0 || len(self.EntityId) == 0 || len(self.MetadataDocument) == 0 {
		return false
	}
	metadata := SamlIdpInstance().GetMetadata(self.Id)
	if self.EntityId != metadata.EntityId {
		return false
	}

	keyword := fmt.Sprintf("login/%s", self.CloudaccountId)
	if strings.Contains(self.MetadataDocument, keyword) {
		return false
	}
	return true
}

func (self *SSAMLProvider) StartSAMLProviderUpdateMetadataTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "SAMLProviderUpdateMetadataTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.SAML_PROVIDER_STATUS_UPDATE_METADATA, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SSAMLProvider) StartSAMLProviderCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "SAMLProviderCreateTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.SAML_PROVIDER_STATUS_CREATING, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SSAMLProvider) StartSAMLProviderSyncTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "SAMLProviderSyncTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.SAML_PROVIDER_STATUS_SYNC, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SSAMLProvider) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.RealDelete(ctx, userCred)
}

func (self *SSAMLProvider) SyncWithCloudSAMLProvider(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudSAMLProvider) error {
	_, err := db.Update(self, func() error {
		self.ExternalId = ext.GetGlobalId()
		self.AuthUrl = ext.GetAuthUrl()
		self.Status = ext.GetStatus()
		metadata, err := ext.GetMetadataDocument()
		if err != nil {
			log.Errorf("failed to get metadata for %s error: %v", self.Name, err)
		}
		if metadata != nil {
			self.EntityId = metadata.EntityId
			self.MetadataDocument = metadata.String()
			if self.IsNeedUpldateMetadata() {
				self.Status = api.SAML_PROVIDER_STATUS_NOT_MATCH
			}
		}
		if self.EntityId != options.Options.ApiServer {
			self.Status = api.SAML_PROVIDER_STATUS_NOT_MATCH
		}
		return nil
	})
	return err
}

func (self *SSAMLProvider) GetMetadataDocument() (samlutils.EntityDescriptor, error) {
	return samlutils.ParseMetadata([]byte(self.MetadataDocument))
}

func (manager *SSAMLProviderManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.SAMLProviderDetails {
	rows := make([]api.SAMLProviderDetails, len(objs))
	infRows := manager.SStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	acRows := manager.SCloudaccountResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.SAMLProviderDetails{
			StatusInfrasResourceBaseDetails: infRows[i],
			CloudaccountResourceDetails:     acRows[i],
		}
	}
	return rows
}

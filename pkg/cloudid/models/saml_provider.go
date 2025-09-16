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
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/samlutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudid/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SSAMLProviderManager struct {
	db.SStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SCloudaccountResourceBaseManager
	SCloudproviderResourceBaseManager
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
	SCloudproviderResourceBase

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

func (self *SSAMLProvider) GetCloudprovider() (*SCloudprovider, error) {
	provider, err := CloudproviderManager.FetchById(self.ManagerId)
	if err != nil {
		return nil, errors.Wrapf(err, "CloudproviderManager.FetchById(%s)", self.ManagerId)
	}
	return provider.(*SCloudprovider), nil
}

func (self *SSAMLProvider) GetProvider() (cloudprovider.ICloudProvider, error) {
	if len(self.ManagerId) > 0 {
		provider, err := self.GetCloudprovider()
		if err != nil {
			return nil, errors.Wrap(err, "GetCloudprovider")
		}
		return provider.GetProvider()
	}
	account, err := self.GetCloudaccount()
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudaccount")
	}
	return account.GetProvider()
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

	q, err = manager.SCloudproviderResourceBaseManager.ListItemFilter(ctx, q, userCred, query.CloudproviderResourceListInput)
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

func (self *SSAMLProvider) StartSAMLProviderDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "SAMLProviderDeleteTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_DELETING, "")
	return task.ScheduleRun(nil)
}

func (self *SSAMLProvider) GetISAMLProvider() (cloudprovider.ICloudSAMLProvider, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	provider, err := self.GetProvider()
	if err != nil {
		return nil, err
	}
	samlProviders, err := provider.GetICloudSAMLProviders()
	if err != nil {
		return nil, err
	}
	for i := range samlProviders {
		if samlProviders[i].GetGlobalId() == self.ExternalId {
			return samlProviders[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, self.ExternalId)
}

func (self *SSAMLProvider) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SSAMLProvider) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SSAMLProvider) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.RealDelete(ctx, userCred)
}

func (self *SSAMLProvider) SyncWithCloudSAMLProvider(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudSAMLProvider, managerId string) error {
	_, err := db.Update(self, func() error {
		self.ExternalId = ext.GetGlobalId()
		self.AuthUrl = ext.GetAuthUrl(options.Options.ApiServer)
		metadata, _ := ext.GetMetadataDocument()
		if metadata != nil {
			self.EntityId = metadata.EntityId
			self.MetadataDocument = metadata.String()
		}
		self.Status = apis.STATUS_UNKNOWN
		if self.EntityId == options.Options.ApiServer && strings.Contains(self.MetadataDocument, "login/"+self.ManagerId) {
			self.Status = apis.STATUS_AVAILABLE
		}
		if len(managerId) > 0 {
			self.ManagerId = managerId
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

func (self *SCloudaccount) SyncSAMLProviders(ctx context.Context, userCred mcclient.TokenCredential, samls []cloudprovider.ICloudSAMLProvider, managerId string) compare.SyncResult {
	result := compare.SyncResult{}
	dbSamls, err := self.GetSAMLProviders(managerId)
	if err != nil {
		result.Error(errors.Wrap(err, "GetSAMLProviders"))
		return result
	}

	removed := make([]SSAMLProvider, 0)
	commondb := make([]SSAMLProvider, 0)
	commonext := make([]cloudprovider.ICloudSAMLProvider, 0)
	added := make([]cloudprovider.ICloudSAMLProvider, 0)

	err = compare.CompareSets(dbSamls, samls, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(errors.Wrap(err, "compare.CompareSets"))
		return result
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].RealDelete(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudSAMLProvider(ctx, userCred, commonext[i], managerId)
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}

	for i := 0; i < len(added); i++ {
		err = self.newFromCloudSAMLProvider(ctx, userCred, added[i], managerId)
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}

func (self *SCloudaccount) newFromCloudSAMLProvider(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudSAMLProvider, managerId string) error {
	saml := &SSAMLProvider{}
	saml.SetModelManager(SAMLProviderManager, saml)
	saml.Name = ext.GetName()
	saml.ExternalId = ext.GetGlobalId()
	saml.DomainId = self.DomainId
	saml.CloudaccountId = self.Id
	saml.ManagerId = managerId
	saml.AuthUrl = ext.GetAuthUrl(options.Options.ApiServer)
	metadata, _ := ext.GetMetadataDocument()
	if metadata != nil {
		saml.EntityId = metadata.EntityId
		saml.MetadataDocument = metadata.String()
	}
	saml.Status = ext.GetStatus()
	saml.Status = apis.STATUS_UNKNOWN
	if saml.EntityId == options.Options.ApiServer && strings.Contains(saml.MetadataDocument, "login/"+saml.ManagerId) {
		saml.Status = apis.STATUS_AVAILABLE
	}
	return SAMLProviderManager.TableSpec().Insert(ctx, saml)
}

func (self *SCloudprovider) GetSamlProviders() ([]SSAMLProvider, error) {
	q := SAMLProviderManager.Query().Equals("manager_id", self.Id)
	ret := []SSAMLProvider{}
	err := db.FetchModelObjects(SAMLProviderManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

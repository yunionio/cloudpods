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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SProjectMappingManager struct {
	db.SEnabledStatusInfrasResourceBaseManager
}

var ProjectMappingManager *SProjectMappingManager

var projectRuleMapping map[string]*SProjectMapping = map[string]*SProjectMapping{}

func init() {
	ProjectMappingManager = &SProjectMappingManager{
		SEnabledStatusInfrasResourceBaseManager: db.NewEnabledStatusInfrasResourceBaseManager(
			SProjectMapping{},
			"project_mappings_tbl",
			"project_mapping",
			"project_mappings",
		),
	}
	ProjectMappingManager.SetVirtualObject(ProjectMappingManager)
}

func GetRuleMapping(id string) (*SProjectMapping, error) {
	rm, ok := projectRuleMapping[id]
	if ok {
		return rm, nil
	}
	pm, err := ProjectMappingManager.FetchById(id)
	if err != nil {
		return nil, errors.Wrapf(err, "ProjectMappingManager.FetchById(%s)", id)
	}
	projectMap := pm.(*SProjectMapping)
	projectRuleMapping[id] = projectMap
	return projectMap, nil
}

type SProjectMapping struct {
	db.SEnabledStatusInfrasResourceBase

	Rules *api.MappingRules `list:"domain" update:"domain" create:"required"`
}

// 列出项目映射表
func (manager *SProjectMappingManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ProjectMappingListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SProjectMappingManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.ProjectMappingCreateInput,
) (api.ProjectMappingCreateInput, error) {
	err := input.Rules.Validate()
	if err != nil {
		return input, err
	}
	var tenant *db.STenant
	for i := range input.Rules {
		if len(input.Rules[i].ProjectId) > 0 {
			projectInput := apis.ProjectizedResourceCreateInput{
				ProjectizedResourceInput: apis.ProjectizedResourceInput{
					ProjectId: input.Rules[i].ProjectId,
				},
			}
			tenant, _, err = db.ValidateProjectizedResourceInput(ctx, projectInput)
			if err != nil {
				return input, err
			}
			input.Rules[i].DomainId = tenant.DomainId
			input.Rules[i].Domain = tenant.Domain
			input.Rules[i].Project = tenant.Name
		}
	}
	input.Rules = input.Rules.Rules()
	input.SetEnabled()
	input.Status = api.PROJECT_MAPPING_STATUS_AVAILABLE
	input.EnabledStatusInfrasResourceBaseCreateInput, err = manager.SEnabledStatusInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusInfrasResourceBaseCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (self *SProjectMapping) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SEnabledStatusInfrasResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	self.refreshMapping()
}

func (manager *SProjectMappingManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ProjectMappingDetails {
	rows := make([]api.ProjectMappingDetails, len(objs))
	stdRows := manager.SEnabledStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	mpIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.ProjectMappingDetails{
			EnabledStatusInfrasResourceBaseDetails: stdRows[i],
		}
		mp := objs[i].(*SProjectMapping)
		mpIds[i] = mp.Id
	}
	type sBind struct {
		Id               string
		Name             string
		ProjectMappingId string
	}
	accounts, managers := []sBind{}, []sBind{}
	q := CloudaccountManager.Query().In("project_mapping_id", mpIds)
	err := q.All(&accounts)
	if err != nil {
		return rows
	}
	q = CloudproviderManager.Query().In("project_mapping_id", mpIds)
	err = q.All(&managers)
	if err != nil {
		return rows
	}
	accountMapping := map[string][]api.SProjectMappingAccount{}
	for i := range accounts {
		_, ok := accountMapping[accounts[i].ProjectMappingId]
		if !ok {
			accountMapping[accounts[i].ProjectMappingId] = []api.SProjectMappingAccount{}
		}
		account := api.SProjectMappingAccount{
			Id:   accounts[i].Id,
			Name: accounts[i].Name,
		}
		accountMapping[accounts[i].ProjectMappingId] = append(accountMapping[accounts[i].ProjectMappingId], account)
	}
	managerMapping := map[string][]api.SProjectMappingAccount{}
	for i := range managers {
		_, ok := managerMapping[managers[i].ProjectMappingId]
		if !ok {
			managerMapping[managers[i].ProjectMappingId] = []api.SProjectMappingAccount{}
		}
		manager := api.SProjectMappingAccount{
			Id:   managers[i].Id,
			Name: managers[i].Name,
		}
		managerMapping[managers[i].ProjectMappingId] = append(managerMapping[managers[i].ProjectMappingId], manager)
	}

	for i := range rows {
		rows[i].Accounts, _ = accountMapping[mpIds[i]]
		rows[i].Managers, _ = managerMapping[mpIds[i]]
	}
	return rows
}

func (manager *SProjectMappingManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SEnabledStatusInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SProjectMappingManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ProjectMappingListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SProjectMappingManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusInfrasResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (self *SProjectMapping) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ProjectMappingUpdateInput) (api.ProjectMappingUpdateInput, error) {
	err := input.Rules.Validate()
	if err != nil {
		return input, err
	}
	var tenant *db.STenant
	for i := range input.Rules {
		if len(input.Rules[i].ProjectId) > 0 {
			projectInput := apis.ProjectizedResourceCreateInput{
				ProjectizedResourceInput: apis.ProjectizedResourceInput{
					ProjectId: input.Rules[i].ProjectId,
				},
			}
			tenant, _, err = db.ValidateProjectizedResourceInput(ctx, projectInput)
			if err != nil {
				return input, err
			}
			input.Rules[i].DomainId = tenant.DomainId
			input.Rules[i].Domain = tenant.Domain
			input.Rules[i].Project = tenant.Name
		}
	}
	input.Rules = input.Rules.Rules()
	input.EnabledStatusInfrasResourceBaseUpdateInput, err = self.SEnabledStatusInfrasResourceBase.ValidateUpdateData(ctx, userCred, query, input.EnabledStatusInfrasResourceBaseUpdateInput)
	return input, err
}

func (self *SProjectMapping) GetCloudaccounts() ([]SCloudaccount, error) {
	q := CloudaccountManager.Query().Equals("project_mapping_id", self.Id)
	accounts := []SCloudaccount{}
	err := db.FetchModelObjects(CloudaccountManager, q, &accounts)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func (self *SProjectMapping) GetCloudproviders() ([]SCloudprovider, error) {
	q := CloudproviderManager.Query().Equals("project_mapping_id", self.Id)
	ret := []SCloudprovider{}
	err := db.FetchModelObjects(CloudproviderManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SProjectMapping) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	delete(projectRuleMapping, self.Id)
	return self.SEnabledStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SProjectMapping) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	accounts, err := self.GetCloudaccounts()
	if err != nil {
		return errors.Wrapf(err, "GetCloudaccounts")
	}
	if len(accounts) > 0 {
		return httperrors.NewNotEmptyError("project mapping has associate %d accounts", len(accounts))
	}
	providers, err := self.GetCloudproviders()
	if err != nil {
		return errors.Wrapf(err, "GetCloudproviders")
	}
	if len(providers) > 0 {
		return httperrors.NewNotEmptyError("project mapping has associate %d cloudproviders", len(providers))
	}
	return self.SEnabledStatusInfrasResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (self *SProjectMapping) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SEnabledStatusInfrasResourceBase.PostUpdate(ctx, userCred, query, data)
	self.refreshMapping()
}

func (self *SProjectMapping) refreshMapping() error {
	projectRuleMapping[self.Id] = self
	return nil
}

// 启用资源映射
func (self *SProjectMapping) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	_, err := self.SEnabledStatusInfrasResourceBase.PerformEnable(ctx, userCred, query, input)
	if err != nil {
		return nil, err
	}
	return nil, self.refreshMapping()
}

// 禁用资源映射
func (self *SProjectMapping) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	_, err := self.SEnabledStatusInfrasResourceBase.PerformDisable(ctx, userCred, query, input)
	if err != nil {
		return nil, err
	}
	return nil, self.refreshMapping()
}

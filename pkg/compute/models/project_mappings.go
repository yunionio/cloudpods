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
			projectInput := apis.ProjectizedResourceInput{ProjectId: input.Rules[i].ProjectId}
			tenant, projectInput, err = db.ValidateProjectizedResourceInput(ctx, projectInput)
			if err != nil {
				return input, err
			}
			input.Rules[i].DomainId = tenant.DomainId
		}
	}
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
	projIds := []string{}
	domainIds := []string{}
	for i := range rows {
		rows[i] = api.ProjectMappingDetails{
			EnabledStatusInfrasResourceBaseDetails: stdRows[i],
		}
		mp := objs[i].(*SProjectMapping)
		mpIds[i] = mp.Id
		if mp.Rules != nil {
			for _, r := range *mp.Rules {
				if len(r.ProjectId) > 0 {
					projIds = append(projIds, r.ProjectId)
					domainIds = append(domainIds, r.DomainId)
				}
			}
		}
	}
	q := db.DefaultDomainQuery("domain", "domain_id").In("domain_id", domainIds)
	domains := []struct {
		DomainId string
		Domain   string
	}{}
	err := q.All(&domainIds)
	if err != nil {
		return rows
	}
	domainMaps := map[string]string{}
	for _, domain := range domains {
		domainMaps[domain.DomainId] = domain.Domain
	}
	q = db.DefaultProjectQuery("id", "name").In("id", projIds)
	projects := []struct {
		Id   string
		Name string
	}{}
	err = q.All(&projects)
	if err != nil {
		return rows
	}
	projectMaps := map[string]string{}
	for _, proj := range projects {
		projectMaps[proj.Id] = proj.Name
	}
	accounts := []struct {
		Id               string
		Name             string
		ProjectMappingId string
	}{}
	q = CloudaccountManager.Query().In("project_mapping_id", mpIds)
	err = q.All(&accounts)
	if err != nil {
		return rows
	}
	accountMapping := map[string][]struct {
		Id   string
		Name string
	}{}
	for i := range accounts {
		_, ok := accountMapping[accounts[i].ProjectMappingId]
		if !ok {
			accountMapping[accounts[i].ProjectMappingId] = []struct {
				Id   string
				Name string
			}{}
		}
		accountMapping[accounts[i].ProjectMappingId] = append(accountMapping[accounts[i].ProjectMappingId],
			struct {
				Id   string
				Name string
			}{
				Id:   accounts[i].Id,
				Name: accounts[i].Name,
			})
	}

	for i := range rows {
		mp := objs[i].(*SProjectMapping)
		if mp.Rules != nil {
			rows[i].Rules = []api.ProjectMappingRuleInfoDetails{}
			rows[i].Accounts, _ = accountMapping[mpIds[i]]
			for i := range *mp.Rules {
				rules := *mp.Rules
				rule := api.ProjectMappingRuleInfoDetails{
					ProjectMappingRuleInfo: rules[i],
				}
				rule.Tenant, _ = projectMaps[rules[i].ProjectId]
				rule.Project = rule.Tenant
				rule.Domain, _ = domainMaps[rules[i].DomainId]
				rows[i].Rules = append(rows[i].Rules, rule)
			}
		}
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
			projectInput := apis.ProjectizedResourceInput{ProjectId: input.Rules[i].ProjectId}
			tenant, projectInput, err = db.ValidateProjectizedResourceInput(ctx, projectInput)
			if err != nil {
				return input, err
			}
			input.Rules[i].DomainId = tenant.DomainId
		}
	}
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

func (self *SProjectMapping) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	delete(projectRuleMapping, self.Id)
	return self.SEnabledStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SProjectMapping) ValidateDeleteCondition(ctx context.Context) error {
	accounts, err := self.GetCloudaccounts()
	if err != nil {
		return errors.Wrapf(err, "GetCloudaccounts")
	}
	if len(accounts) > 0 {
		return httperrors.NewNotEmptyError("project mapping has associate %d accounts", len(accounts))
	}
	return self.SEnabledStatusInfrasResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SProjectMapping) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SEnabledStatusInfrasResourceBase.PostUpdate(ctx, userCred, query, data)
	self.refreshMapping()
}

func (self *SProjectMapping) refreshMapping() error {
	projectRuleMapping[self.Id] = self
	return nil
}

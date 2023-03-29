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
	"database/sql"
	"fmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SExternalProjectManager struct {
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
}

var ExternalProjectManager *SExternalProjectManager

func init() {
	ExternalProjectManager = &SExternalProjectManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SExternalProject{},
			"externalprojects_tbl",
			"externalproject",
			"externalprojects",
		),
	}
	ExternalProjectManager.SetVirtualObject(ExternalProjectManager)
}

type SExternalProject struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase
	SManagedResourceBase

	ExternalDomainId string `width:"36" charset:"ascii" nullable:"true" list:"user"`
	// 归属云账号ID
	CloudaccountId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
}

func (manager *SExternalProjectManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.ExternalProjectCreateInput,
) (api.ExternalProjectCreateInput, error) {
	var err error
	if len(input.CloudaccountId) == 0 {
		return input, httperrors.NewMissingParameterError("cloudaccount_id")
	}
	_account, err := validators.ValidateModel(userCred, CloudaccountManager, &input.CloudaccountId)
	if err != nil {
		return input, err
	}
	account := _account.(*SCloudaccount)
	driver, err := account.GetProvider(ctx)
	if err != nil {
		return input, err
	}
	if utils.IsInStringArray(account.Provider, api.MANGER_EXTERNAL_PROJECT_PROVIDERS) {
		if len(input.ManagerId) == 0 {
			return input, httperrors.NewMissingParameterError("manager_id")
		}
		_provider, err := validators.ValidateModel(userCred, CloudproviderManager, &input.ManagerId)
		if err != nil {
			return input, err
		}
		provider := _provider.(*SCloudprovider)
		driver, err = provider.GetProvider(ctx)
		if err != nil {
			return input, err
		}
	}

	input.VirtualResourceCreateInput, err = manager.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SVirtualResourceBaseManager.ValidateCreateData")
	}

	iProj, err := driver.CreateIProject(input.Name)
	if err != nil {
		return input, errors.Wrapf(err, "CreateIProject")
	}
	input.ExternalId = iProj.GetGlobalId()
	input.Status = api.EXTERNAL_PROJECT_STATUS_AVAILABLE
	return input, nil
}

func (manager *SExternalProjectManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ExternalProjectDetails {
	rows := make([]api.ExternalProjectDetails, len(objs))
	virRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	accountIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.ExternalProjectDetails{
			VirtualResourceDetails: virRows[i],
			ManagedResourceInfo:    managerRows[i],
		}
		proj := objs[i].(*SExternalProject)
		accountIds[i] = proj.CloudaccountId
	}
	accounts := make(map[string]SCloudaccount)
	err := db.FetchStandaloneObjectsByIds(CloudaccountManager, accountIds, &accounts)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds (%s) fail %s",
			CloudaccountManager.KeywordPlural(), err)
		return rows
	}

	for i := range rows {
		if account, ok := accounts[accountIds[i]]; ok {
			rows[i].Account = account.Name
			rows[i].Brand = account.Brand
			rows[i].Provider = account.Provider
		}
	}

	return rows
}

func (manager *SExternalProjectManager) GetProject(externalId string, providerId string) (*SExternalProject, error) {
	project := &SExternalProject{}
	project.SetModelManager(manager, project)
	sq := CloudproviderManager.Query("cloudaccount_id").Equals("id", providerId)
	q := manager.Query().Equals("external_id", externalId).Equals("cloudaccount_id", sq.SubQuery())
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, fmt.Errorf("no external project record %s for provider %s", externalId, providerId)
	}
	if count > 1 {
		return nil, fmt.Errorf("duplicate external project record %s for provider %s", externalId, providerId)
	}
	return project, q.First(project)
}

func (manager *SExternalProjectManager) SyncProjects(ctx context.Context, userCred mcclient.TokenCredential, account *SCloudaccount, projects []cloudprovider.ICloudProject, xor bool) compare.SyncResult {
	lockman.LockRawObject(ctx, manager.Keyword(), account.Id)
	defer lockman.ReleaseRawObject(ctx, manager.Keyword(), account.Id)

	syncResult := compare.SyncResult{}

	dbProjects, err := account.GetExternalProjects()
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := make([]SExternalProject, 0)
	commondb := make([]SExternalProject, 0)
	commonext := make([]cloudprovider.ICloudProject, 0)
	added := make([]cloudprovider.ICloudProject, 0)

	err = compare.CompareSets(dbProjects, projects, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].syncRemoveCloudProject(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	if !xor {
		for i := 0; i < len(commondb); i++ {
			err = commondb[i].SyncWithCloudProject(ctx, userCred, account, commonext[i])
			if err != nil {
				syncResult.UpdateError(err)
			} else {
				syncResult.Update()
			}
		}
	}
	for i := 0; i < len(added); i++ {
		_, err := manager.newFromCloudProject(ctx, userCred, account, nil, added[i])
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncResult.Add()
		}
	}
	return syncResult
}

func (self *SExternalProject) syncRemoveCloudProject(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	return self.Delete(ctx, userCred)
}

func (self *SExternalProject) SyncWithCloudProject(ctx context.Context, userCred mcclient.TokenCredential, account *SCloudaccount, ext cloudprovider.ICloudProject) error {
	s := auth.GetAdminSession(ctx, consts.GetRegion())
	providers := account.GetCloudproviders()
	providerMaps := map[string]string{}
	for _, provider := range providers {
		providerMaps[provider.Account] = provider.Id
	}
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Name = ext.GetName()
		self.IsEmulated = ext.IsEmulated()
		self.Status = ext.GetStatus()
		if accountId := ext.GetAccountId(); len(accountId) > 0 {
			self.ManagerId, _ = providerMaps[accountId]
		}
		share := account.GetSharedInfo()
		if self.DomainId != account.DomainId && !(share.PublicScope == rbacscope.ScopeSystem ||
			(share.PublicScope == rbacscope.ScopeDomain && utils.IsInStringArray(self.DomainId, share.SharedDomains))) {
			self.ProjectId = account.ProjectId
			self.DomainId = account.DomainId
			if account.AutoCreateProject {
				desc := fmt.Sprintf("auto create from cloud project %s (%s)", self.Name, self.ExternalId)
				domainId, projectId, err := account.getOrCreateTenant(ctx, self.Name, "", "", desc)
				if err != nil {
					log.Errorf("failed to get or create tenant %s(%s) %v", self.Name, self.ExternalId, err)
				} else {
					self.ProjectId = projectId
					self.DomainId = domainId
				}
			}
			return nil
		}
		pm, _ := account.GetProjectMapping()
		if pm != nil && pm.Enabled.IsTrue() && pm.IsNeedProjectSync() && self.ProjectSrc != string(apis.OWNER_SOURCE_LOCAL) {
			extTags, err := ext.GetTags()
			if err != nil {
				return errors.Wrapf(err, "extModel.GetTags")
			}
			find := false
			if pm.Rules != nil {
				for _, rule := range *pm.Rules {
					domainId, projectId, newProj, isMatch := rule.IsMatchTags(extTags)
					if isMatch && len(newProj) > 0 {
						domainId, projectId, err = account.getOrCreateTenant(context.TODO(), newProj, "", "", "auto create from tag")
						if err != nil {
							log.Errorf("getOrCreateTenant(%s) error: %v", newProj, err)
							continue
						}
						if len(domainId) > 0 && len(projectId) > 0 {
							self.DomainId = domainId
							self.ProjectId = projectId
							find = true
							break
						}
					}
				}
			}
			if !find && account.AutoCreateProject {
				desc := fmt.Sprintf("auto create from cloud project %s (%s)", self.Name, self.ExternalId)
				domainId, projectId, err := account.getOrCreateTenant(ctx, self.Name, self.DomainId, "", desc)
				if err != nil {
					log.Errorf("failed to get or create tenant %s(%s) %v", self.Name, self.ExternalId, err)
				} else {
					self.DomainId = domainId
					self.ProjectId = projectId
				}
			}
		} else if account.AutoCreateProject && options.Options.EnableAutoRenameProject {
			tenant, err := db.TenantCacheManager.FetchTenantById(ctx, self.ProjectId)
			if err != nil {
				return errors.Wrapf(err, "TenantCacheManager.FetchTenantById(%s)", self.ProjectId)
			}
			if tenant.Name != self.Name {
				proj, err := db.TenantCacheManager.FetchTenantByNameInDomain(ctx, self.Name, tenant.DomainId)
				if err != nil {
					if errors.Cause(err) == sql.ErrNoRows {
						params := map[string]string{"name": self.Name}
						_, err := identity.Projects.Update(s, tenant.Id, jsonutils.Marshal(params))
						if err != nil {
							return errors.Wrapf(err, "update project name from %s -> %s", tenant.Name, self.Name)
						}
						_, err = db.Update(tenant, func() error {
							tenant.Name = self.Name
							return nil
						})
						return err
					}
					return errors.Wrapf(err, "FetchTenantByName(%s)", self.Name)
				}
				if proj.DomainId == account.DomainId ||
					share.PublicScope == rbacscope.ScopeSystem ||
					(share.PublicScope == rbacscope.ScopeDomain && utils.IsInStringArray(proj.DomainId, share.SharedDomains)) {
					self.ProjectId = proj.Id
					self.DomainId = proj.DomainId
					return nil
				}
			}
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.UpdateWithLock")
	}

	tags, _ := ext.GetTags()
	if len(tags) > 0 {
		identity.Projects.PerformAction(s, self.ProjectId, "user-metadata", jsonutils.Marshal(tags))
	}
	syncMetadata(ctx, userCred, self, ext)
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (self *SCloudaccount) getOrCreateDomain(ctx context.Context, userCred mcclient.TokenCredential, id, name string) (string, error) {
	lockman.LockRawObject(ctx, self.Id, CloudaccountManager.Keyword())
	defer lockman.ReleaseRawObject(ctx, self.Id, CloudaccountManager.Keyword())

	domainId := ""
	domain, err := db.TenantCacheManager.FetchDomainByIdOrName(ctx, name)
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			return "", errors.Wrapf(err, "FetchDomainByIdOrName")
		}
		s := auth.GetAdminSession(ctx, options.Options.Region)
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(name), "generate_name")

		desc := fmt.Sprintf("auto create from cloud project %s (%s)", id, name)
		params.Add(jsonutils.NewString(desc), "description")

		resp, err := identity.Domains.Create(s, params)
		if err != nil {
			return "", errors.Wrap(err, "Projects.Create")
		}
		domainId, err = resp.GetString("id")
		if err != nil {
			return "", errors.Wrapf(err, "resp.GetString")
		}
	} else {
		domainId = domain.Id
	}

	share := self.GetSharedInfo()
	if share.PublicScope == rbacscope.ScopeSystem {
		return domainId, nil
	}
	input := api.CloudaccountPerformPublicInput{}
	input.ShareMode = string(rbacscope.ScopeSystem)
	input.PerformPublicDomainInput = apis.PerformPublicDomainInput{
		Scope:           string(rbacscope.ScopeDomain),
		SharedDomains:   append(share.SharedDomains, domainId),
		SharedDomainIds: append(share.SharedDomains, domainId),
	}
	_, err = self.SInfrasResourceBase.PerformPublic(ctx, userCred, jsonutils.NewDict(), input.PerformPublicDomainInput)
	if err != nil {
		return "", errors.Wrapf(err, "PerformPublic")
	}
	return domainId, self.setShareMode(userCred, input.ShareMode)
}

func (manager *SExternalProjectManager) newFromCloudProject(ctx context.Context, userCred mcclient.TokenCredential, account *SCloudaccount, localProject *db.STenant, extProject cloudprovider.ICloudProject) (*SExternalProject, error) {
	project := SExternalProject{}
	project.SetModelManager(manager, &project)

	project.Name = extProject.GetName()
	project.Status = extProject.GetStatus()
	project.ExternalId = extProject.GetGlobalId()
	project.IsEmulated = extProject.IsEmulated()
	project.CloudaccountId = account.Id
	project.DomainId = account.DomainId
	project.ProjectId = account.ProjectId
	project.ExternalDomainId = extProject.GetDomainId()

	providers := account.GetCloudproviders()
	providerMaps := map[string]string{}
	for _, provider := range providers {
		providerMaps[provider.Account] = provider.Id
	}
	if accountId := extProject.GetAccountId(); len(accountId) > 0 {
		project.ManagerId, _ = providerMaps[accountId]
	}

	domainName := extProject.GetDomainName()
	if len(project.ExternalDomainId) > 0 && len(domainName) > 0 {
		domainId, err := account.getOrCreateDomain(ctx, userCred, project.ExternalDomainId, domainName)
		if err != nil {
			log.Errorf("getOrCreateDomain for project %s error: %v", account.Name, err)
		} else {
			project.DomainId = domainId
		}
	}
	pm, _ := account.GetProjectMapping()
	if localProject != nil {
		project.DomainId = localProject.DomainId
		project.ProjectId = localProject.Id
	} else if pm != nil && pm.Enabled.IsTrue() && pm.IsNeedProjectSync() {
		extTags, err := extProject.GetTags()
		if err != nil {
			return nil, errors.Wrapf(err, "extModel.GetTags")
		}
		find := false
		if pm.Rules != nil {
			for _, rule := range *pm.Rules {
				domainId, projectId, newProj, isMatch := rule.IsMatchTags(extTags)
				if isMatch && len(newProj) > 0 {
					domainId, projectId, err = account.getOrCreateTenant(context.TODO(), newProj, "", "", "auto create from tag")
					if err != nil {
						log.Errorf("getOrCreateTenant(%s) error: %v", newProj, err)
						continue
					}
					if len(domainId) > 0 && len(projectId) > 0 {
						project.DomainId = domainId
						project.ProjectId = projectId
						find = true
						break
					}
				}
			}
		}
		if !find && account.AutoCreateProject {
			desc := fmt.Sprintf("auto create from cloud project %s (%s)", project.Name, project.ExternalId)
			domainId, projectId, err := account.getOrCreateTenant(ctx, project.Name, project.DomainId, "", desc)
			if err != nil {
				log.Errorf("failed to get or create tenant %s(%s) %v", project.Name, project.ExternalId, err)
			} else {
				project.DomainId = domainId
				project.ProjectId = projectId
			}
		}
	} else if account.AutoCreateProject {
		desc := fmt.Sprintf("auto create from cloud project %s (%s)", project.Name, project.ExternalId)
		domainId, projectId, err := account.getOrCreateTenant(ctx, project.Name, project.DomainId, "", desc)
		if err != nil {
			log.Errorf("failed to get or create tenant %s(%s) %v", project.Name, project.ExternalId, err)
		} else {
			project.DomainId = domainId
			project.ProjectId = projectId
		}
	}

	err := manager.TableSpec().Insert(ctx, &project)
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	tags, _ := extProject.GetTags()
	if len(tags) > 0 {
		s := auth.GetAdminSession(ctx, consts.GetRegion())
		identity.Projects.PerformAction(s, project.ProjectId, "user-metadata", jsonutils.Marshal(tags))
	}

	syncMetadata(ctx, userCred, &project, extProject)
	db.OpsLog.LogEvent(&project, db.ACT_CREATE, project.GetShortDesc(ctx), userCred)
	return &project, nil
}

func (self *SExternalProject) GetCloudaccount() (*SCloudaccount, error) {
	account, err := CloudaccountManager.FetchById(self.CloudaccountId)
	if err != nil {
		return nil, errors.Wrapf(err, "CloudaccountManager.FetchById(%s)", self.CloudaccountId)
	}
	return account.(*SCloudaccount), nil
}

// 切换本地项目
func (self *SExternalProject) PerformChangeProject(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ExternalProjectChangeProjectInput) (jsonutils.JSONObject, error) {
	if len(input.ProjectId) == 0 {
		return nil, httperrors.NewMissingParameterError("project_id")
	}

	tenant, err := db.TenantCacheManager.FetchTenantByIdOrNameInDomain(ctx, input.ProjectId, input.ProjectDomainId)
	if err != nil {
		return nil, httperrors.NewNotFoundError("project %s not found", input.ProjectId)
	}

	if self.ProjectId == tenant.Id {
		return nil, nil
	}

	account, err := self.GetCloudaccount()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetCloudaccount"))
	}
	share := account.GetSharedInfo()

	if self.DomainId != tenant.DomainId && !(tenant.DomainId == account.DomainId || share.PublicScope == rbacscope.ScopeSystem ||
		(share.PublicScope == rbacscope.ScopeDomain && utils.IsInStringArray(tenant.DomainId, share.SharedDomains))) {
		return nil, httperrors.NewForbiddenError("account %s not share for domain %s", account.Name, tenant.DomainId)
	}

	notes := struct {
		OldProjectId string
		OldDomainId  string
		NewProjectId string
		NewProject   string
		NewDomainId  string
		NewDomain    string
	}{
		OldProjectId: self.ProjectId,
		OldDomainId:  self.DomainId,
		NewProjectId: tenant.Id,
		NewProject:   tenant.Name,
		NewDomainId:  tenant.DomainId,
		NewDomain:    tenant.Domain,
	}

	_, err = db.Update(self, func() error {
		self.ProjectId = tenant.Id
		self.DomainId = tenant.DomainId
		return nil
	})
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "db.Update"))
	}

	logclient.AddSimpleActionLog(self, logclient.ACT_CHANGE_OWNER, notes, userCred, true)
	return nil, nil
}

// 云平台导入项目列表
func (manager *SExternalProjectManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ExternalProjectListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	if len(query.CloudproviderId) > 0 {
		p, err := CloudproviderManager.FetchByIdOrName(userCred, query.CloudproviderId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("cloudprovider", query.CloudproviderId)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		q = q.Equals("manager_id", p.GetId())
		provider := p.(*SCloudprovider)
		query.CloudaccountId = []string{provider.CloudaccountId}
	}

	if len(query.CloudaccountId) > 0 {
		accountIds := []string{}
		for _, _account := range query.CloudaccountId {
			account, err := CloudaccountManager.FetchByIdOrName(userCred, _account)
			if err != nil {
				if errors.Cause(err) == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError2("cloudaccount", _account)
				}
				return nil, httperrors.NewGeneralError(err)
			}
			accountIds = append(accountIds, account.GetId())
		}
		q = q.In("cloudaccount_id", accountIds)
	}

	return q, nil
}

func (manager *SExternalProjectManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ExternalProjectListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SExternalProjectManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SExternalProjectManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (manager *SExternalProjectManager) InitializeData() error {
	eps := []SExternalProject{}
	q := manager.Query().IsNullOrEmpty("cloudaccount_id")
	err := db.FetchModelObjects(manager, q, &eps)
	if err != nil {
		return err
	}
	for i := range eps {
		_, err = db.Update(&eps[i], func() error {
			provider := eps[i].GetCloudprovider()
			if provider == nil {
				return fmt.Errorf("failed to get external project %s cloudprovider", eps[i].Id)
			}
			eps[i].CloudaccountId = provider.CloudaccountId
			return nil
		})
	}
	return nil
}

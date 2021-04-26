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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
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

	// 归属云账号ID
	CloudaccountId string `width:"36" charset:"ascii" nullable:"false" list:"user"`
}

func (manager *SExternalProjectManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsDomainAllowList(userCred, manager)
}

func (self *SExternalProject) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return false
}

func (self *SExternalProject) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.ExternalProjectDetails, error) {
	return api.ExternalProjectDetails{}, nil
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
	accountIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.ExternalProjectDetails{
			VirtualResourceDetails: virRows[i],
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

func (manager *SExternalProjectManager) SyncProjects(ctx context.Context, userCred mcclient.TokenCredential, account *SCloudaccount, projects []cloudprovider.ICloudProject) compare.SyncResult {
	lockman.LockRawObject(ctx, "external-projects", account.Id)
	defer lockman.ReleaseRawObject(ctx, "external-projects", account.Id)

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
	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudProject(ctx, userCred, account, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncResult.Update()
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
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Name = ext.GetName()
		self.IsEmulated = ext.IsEmulated()
		self.Status = ext.GetStatus()
		share := account.GetSharedInfo()
		if self.DomainId != account.DomainId && !(share.PublicScope == rbacutils.ScopeSystem ||
			(share.PublicScope == rbacutils.ScopeDomain && utils.IsInStringArray(self.DomainId, share.SharedDomains))) {
			self.ProjectId = account.ProjectId
			self.DomainId = account.DomainId
			if account.AutoCreateProject {
				desc := fmt.Sprintf("auto create from cloud project %s (%s)", self.Name, self.ExternalId)
				domainId, projectId, err := account.getOrCreateTenant(ctx, self.Name, "", desc)
				if err != nil {
					log.Errorf("failed to get or create tenant %s(%s) %v", self.Name, self.ExternalId, err)
				} else {
					self.ProjectId = projectId
					self.DomainId = domainId
				}
			}
			return nil
		}
		if account.AutoCreateProject && options.Options.EnableAutoRenameProject {
			tenant, err := db.TenantCacheManager.FetchTenantById(ctx, self.ProjectId)
			if err != nil {
				return errors.Wrapf(err, "TenantCacheManager.FetchTenantById(%s)", self.ProjectId)
			}
			if tenant.Name != self.Name {
				proj, err := db.TenantCacheManager.FetchTenantByName(ctx, self.Name)
				if err != nil {
					if errors.Cause(err) == sql.ErrNoRows {
						s := auth.GetAdminSession(ctx, consts.GetRegion(), "v1")
						params := map[string]string{"name": self.Name}
						_, err := modules.Projects.Update(s, tenant.Id, jsonutils.Marshal(params))
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
					share.PublicScope == rbacutils.ScopeSystem ||
					(share.PublicScope == rbacutils.ScopeDomain && utils.IsInStringArray(proj.DomainId, share.SharedDomains)) {
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
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
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
	if localProject != nil {
		project.DomainId = localProject.DomainId
		project.ProjectId = localProject.Id
	} else if account.AutoCreateProject {
		desc := fmt.Sprintf("auto create from cloud project %s (%s)", project.Name, project.ExternalId)
		domainId, projectId, err := account.getOrCreateTenant(ctx, project.Name, "", desc)
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

	db.OpsLog.LogEvent(&project, db.ACT_CREATE, project.GetShortDesc(ctx), userCred)
	return &project, nil
}

func (self *SExternalProject) AllowPerformChangeProject(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "change-project")
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

	tenant, err := db.TenantCacheManager.FetchTenantByIdOrName(ctx, input.ProjectId)
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

	if self.DomainId != tenant.DomainId && !(tenant.DomainId == account.DomainId || share.PublicScope == rbacutils.ScopeSystem ||
		(share.PublicScope == rbacutils.ScopeDomain && utils.IsInStringArray(tenant.DomainId, share.SharedDomains))) {
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

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
	"time"

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
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=externalproject
// +onecloud:swagger-gen-model-plural=externalprojects
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
	// 优先级，同一个本地项目映射多个云上项目，优先级高的优先选择
	// 数值越高，优先级越大
	Priority int `default:"0" list:"user" update:"user" list:"user"`

	// swagger: ignore
	// 将在3.12之后版本移除
	CloudaccountId string `width:"36" charset:"ascii" nullable:"true"`
}

func (manager *SExternalProjectManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.ExternalProjectCreateInput,
) (api.ExternalProjectCreateInput, error) {
	_, err := validators.ValidateModel(ctx, userCred, CloudproviderManager, &input.ManagerId)
	if err != nil {
		return input, err
	}

	input.VirtualResourceCreateInput, err = manager.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SVirtualResourceBaseManager.ValidateCreateData")
	}

	// check duplicity
	exist, err := manager.recordExists(input.ManagerId, input.Name, ownerId.GetProjectId())
	if err != nil {
		return input, errors.Wrap(err, "recordExits")
	} else if exist {
		return input, errors.Wrapf(httperrors.ErrDuplicateResource, "manager_id: %s name: %s project: %s", input.ManagerId, input.Name, ownerId.GetProjectId())
	}
	return input, nil
}

func (manager *SExternalProjectManager) recordExists(managerId, name, projectId string) (bool, error) {
	q := manager.Query()
	q = q.Equals("name", name)
	q = q.Equals("tenant_id", projectId)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	cnt, err := q.CountWithError()
	if err != nil {
		return false, errors.Wrap(err, "CountWithError")
	}
	return cnt > 0, nil
}

func (extProj *SExternalProject) RemoteCreateProject(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := extProj.remoteCreateProjectInternal(ctx, userCred)
	if err == nil {
		return nil
	}
	extProj.SetStatus(ctx, userCred, api.EXTERNAL_PROJECT_STATUS_UNAVAILABLE, err.Error())

	return errors.Wrap(err, "remoteCreateProjectInternal")
}

func (extProj *SExternalProject) remoteCreateProjectInternal(ctx context.Context, userCred mcclient.TokenCredential) error {
	provider, err := extProj.GetCloudprovider()
	if err != nil {
		return errors.Wrap(err, "GetCloudaccount")
	}
	driver, err := provider.GetProvider(ctx)
	if err != nil {
		return errors.Wrap(err, "account.GetProvider")
	}
	iProj, err := driver.CreateIProject(extProj.Name)
	if err != nil {
		return errors.Wrapf(err, "driver.CreateIProject")
	}
	_, err = db.Update(extProj, func() error {
		extProj.ExternalId = iProj.GetGlobalId()
		extProj.Status = api.EXTERNAL_PROJECT_STATUS_AVAILABLE
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "Update Status")
	}

	db.OpsLog.LogEvent(extProj, db.ACT_UPDATE, extProj.GetShortDesc(ctx), userCred)
	logclient.AddActionLogWithContext(ctx, extProj, logclient.ACT_UPDATE, extProj.GetShortDesc(ctx), userCred, true)

	return nil
}

func (extProj *SExternalProject) CustomizeCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) error {
	return extProj.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (extProj *SExternalProject) PostCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) {
	extProj.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	extProj.startExternalProjectCreateTask(ctx, userCred)
}

func (extProj *SExternalProject) startExternalProjectCreateTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	extProj.SetStatus(ctx, userCred, api.EXTERNAL_PROJECT_STATUS_CREATING, "")
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "ExternalProjectCreateTask", extProj, userCred, params, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
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
	for i := range rows {
		rows[i] = api.ExternalProjectDetails{
			VirtualResourceDetails: virRows[i],
			ManagedResourceInfo:    managerRows[i],
		}
	}
	return rows
}

func (manager *SExternalProjectManager) GetProject(externalId string, providerId string) (*SExternalProject, error) {
	project := &SExternalProject{}
	project.SetModelManager(manager, project)
	q := manager.Query().Equals("external_id", externalId).Equals("manager_id", providerId)
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

func (cp *SCloudprovider) GetExternalProjects() ([]SExternalProject, error) {
	projects := []SExternalProject{}
	q := ExternalProjectManager.Query().Equals("manager_id", cp.Id)
	err := db.FetchModelObjects(ExternalProjectManager, q, &projects)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return projects, nil
}

func (cp *SCloudprovider) SyncProjects(ctx context.Context, userCred mcclient.TokenCredential, projects []cloudprovider.ICloudProject, xor bool) compare.SyncResult {
	lockman.LockRawObject(ctx, ExternalProjectManager.Keyword(), cp.Id)
	defer lockman.ReleaseRawObject(ctx, ExternalProjectManager.Keyword(), cp.Id)

	syncResult := compare.SyncResult{}

	dbProjects, err := cp.GetExternalProjects()
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
		if removed[i].Source == apis.EXTERNAL_RESOURCE_SOURCE_LOCAL {
			removed[i].SetStatus(ctx, userCred, api.EXTERNAL_PROJECT_STATUS_UNKNOWN, "sync delete")
		} else {
			err = removed[i].syncRemoveCloudProject(ctx, userCred)
			if err != nil {
				syncResult.DeleteError(err)
			} else {
				syncResult.Delete()
			}
		}
	}
	if !xor {
		for i := 0; i < len(commondb); i++ {
			err = commondb[i].SyncWithCloudProject(ctx, userCred, cp, commonext[i])
			if err != nil {
				syncResult.UpdateError(err)
			} else {
				syncResult.Update()
			}
		}
	}
	for i := 0; i < len(added); i++ {
		_, err := cp.newFromCloudProject(ctx, userCred, nil, added[i])
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

	err := func() error {
		account, err := self.GetCloudaccount()
		if err != nil {
			return errors.Wrapf(err, "GetCloudaccount")
		}
		if !account.AutoCreateProject || !options.Options.EnableAutoRenameProject {
			return nil
		}
		pm, _ := account.GetProjectMapping()
		if pm != nil {
			return nil
		}
		count, _ := self.GetProjectCount()
		if count != 1 {
			return nil
		}
		s := auth.GetAdminSession(ctx, consts.GetRegion())
		_, err = identity.Projects.Delete(s, self.ProjectId, nil)
		if err != nil {
			return errors.Wrapf(err, "try auto delete project %s error: %v", self.Name, err)
		}
		return nil
	}()
	if err != nil {
		log.Errorf("syncRemoveCloudProject %s(%s) error: %v", self.Name, self.Id, err)
	}

	return self.Delete(ctx, userCred)
}

func (self *SExternalProject) GetProjectCount() (int, error) {
	return ExternalProjectManager.Query().Equals("tenant_id", self.ProjectId).CountWithError()
}

func (self *SExternalProject) IsMaxPriority() bool {
	project := &SExternalProject{}
	err := ExternalProjectManager.Query().Equals("tenant_id", self.ProjectId).Desc("priority").First(project)
	if err != nil {
		return false
	}
	return project.Priority == self.Priority
}

func (self *SExternalProject) SyncWithCloudProject(ctx context.Context, userCred mcclient.TokenCredential, cp *SCloudprovider, ext cloudprovider.ICloudProject) error {
	s := auth.GetAdminSession(ctx, consts.GetRegion())
	account, err := cp.GetCloudaccount()
	if err != nil {
		return errors.Wrapf(err, "GetCloudaccount")
	}

	domainId := ""
	projectId := ""
	share := account.GetSharedInfo()
	if self.DomainId != account.DomainId && !(share.PublicScope == rbacscope.ScopeSystem ||
		(share.PublicScope == rbacscope.ScopeDomain && utils.IsInStringArray(self.DomainId, share.SharedDomains))) {
		projectId = account.ProjectId
		domainId = account.DomainId
		if account.AutoCreateProject {
			desc := fmt.Sprintf("auto create from cloud project %s (%s)", self.Name, self.ExternalId)
			var err error
			domainId, projectId, err = account.getOrCreateTenant(ctx, self.Name, "", "", desc)
			if err != nil {
				return errors.Wrapf(err, "getOrCreateTenant")
			}
		}
		return nil
	}

	pm, _ := account.GetProjectMapping()
	if self.ProjectSrc != string(apis.OWNER_SOURCE_LOCAL) {
		find := false
		if pm != nil && pm.Enabled.IsTrue() && pm.IsNeedProjectSync() {
			extTags, err := ext.GetTags()
			if err != nil {
				return errors.Wrapf(err, "extModel.GetTags")
			}
			if pm.Rules != nil {
				for _, rule := range *pm.Rules {
					var newProj string
					var isMatch bool
					domainId, projectId, newProj, isMatch = rule.IsMatchTags(extTags)
					if isMatch && len(newProj) > 0 {
						domainId, projectId, err = account.getOrCreateTenant(ctx, newProj, "", "", "auto create from tag")
						if err != nil {
							log.Errorf("getOrCreateTenant(%s) error: %v", newProj, err)
							continue
						}
						find = true
						break
					}
				}
			}
		}
		if !find && account.AutoCreateProject {
			domainId, projectId = account.DomainId, account.ProjectId
			desc := fmt.Sprintf("auto create from cloud project %s (%s)", self.Name, self.ExternalId)
			var err error
			domainId, projectId, err = account.getOrCreateTenant(ctx, self.Name, self.DomainId, "", desc)
			if err != nil {
				return errors.Wrapf(err, "getOrCreateTenant")
			}
		}
	}

	oldName := self.Name
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Name = ext.GetName()
		self.IsEmulated = ext.IsEmulated()
		self.Status = ext.GetStatus()
		if len(domainId) > 0 && len(projectId) > 0 {
			self.DomainId = domainId
			self.ProjectId = projectId
		}
		cache, err := db.TenantCacheManager.FetchTenantByIdOrNameInDomain(ctx, self.ProjectId, self.DomainId)
		if err != nil {
			return errors.Wrapf(err, "FetchProject %s", self.ProjectId)
		}
		if cache.PendingDeleted {
			desc := fmt.Sprintf("auto create from cloud project %s (%s)", self.Name, self.ExternalId)
			_, self.ProjectId, err = account.getOrCreateTenant(ctx, self.Name, self.DomainId, "", desc)
			if err != nil {
				return errors.Wrapf(err, "getOrCreateTenant")
			}
			return nil
		}
		if pm == nil && account.AutoCreateProject && options.Options.EnableAutoRenameProject && oldName != self.Name {
			count, _ := self.GetProjectCount()
			if count == 1 {
				params := map[string]string{"name": self.Name}
				_, err = identity.Projects.Update(s, self.ProjectId, jsonutils.Marshal(params))
				if err != nil {
					return errors.Wrapf(err, "update project name from %s -> %s", oldName, self.Name)
				}
				_, err = db.Update(cache, func() error {
					cache.Name = self.Name
					return nil
				})
				return err
			}
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.UpdateWithLock")
	}

	if self.IsMaxPriority() {
		tags, _ := ext.GetTags()
		if len(tags) > 0 {
			identity.Projects.PerformAction(s, self.ProjectId, "user-metadata", jsonutils.Marshal(tags))
		}
	}

	syncMetadata(ctx, userCred, self, ext, account.ReadOnly)
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (account *SCloudaccount) getOrCreateTenant(ctx context.Context, name, domainId, projectId, desc string) (string, string, error) {
	if len(domainId) == 0 {
		domainId = account.DomainId
	}

	ctx = context.WithValue(ctx, time.Now().String(), utils.GenRequestId(20))
	lockman.LockRawObject(ctx, domainId, name)
	defer lockman.ReleaseRawObject(ctx, domainId, name)

	tenant, err := getTenant(ctx, projectId, name, domainId)
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			return "", "", errors.Wrapf(err, "getTenan")
		}
		return createTenant(ctx, name, domainId, desc)
	}
	if tenant.PendingDeleted {
		return createTenant(ctx, name, domainId, desc)
	}
	share := account.GetSharedInfo()
	if tenant.DomainId == account.DomainId || (share.PublicScope == rbacscope.ScopeSystem ||
		(share.PublicScope == rbacscope.ScopeDomain && utils.IsInStringArray(tenant.DomainId, share.SharedDomains))) {
		return tenant.DomainId, tenant.Id, nil
	}
	return createTenant(ctx, name, domainId, desc)
}

func (cp *SCloudprovider) newFromCloudProject(ctx context.Context, userCred mcclient.TokenCredential, localProject *db.STenant, extProject cloudprovider.ICloudProject) (*SExternalProject, error) {
	project := SExternalProject{}
	project.SetModelManager(ExternalProjectManager, &project)

	project.Name = extProject.GetName()
	project.Status = extProject.GetStatus()
	project.ExternalId = extProject.GetGlobalId()
	project.IsEmulated = extProject.IsEmulated()
	project.ManagerId = cp.Id
	project.DomainId = cp.DomainId
	project.ProjectId = cp.ProjectId
	project.ProjectSrc = string(apis.OWNER_SOURCE_CLOUD)

	account, err := cp.GetCloudaccount()
	if err != nil {
		return nil, errors.Wrapf(err, "GetCloudaccount")
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

	err = ExternalProjectManager.TableSpec().Insert(ctx, &project)
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	if project.IsMaxPriority() {
		tags, _ := extProject.GetTags()
		if len(tags) > 0 {
			s := auth.GetAdminSession(ctx, consts.GetRegion())
			identity.Projects.PerformAction(s, project.ProjectId, "user-metadata", jsonutils.Marshal(tags))
		}
	}

	syncMetadata(ctx, userCred, &project, extProject, account.ReadOnly)
	db.OpsLog.LogEvent(&project, db.ACT_CREATE, project.GetShortDesc(ctx), userCred)
	return &project, nil
}

func (self *SExternalProject) GetCloudprovider() (*SCloudprovider, error) {
	if len(self.ManagerId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty manager_id")
	}
	provider, err := CloudproviderManager.FetchById(self.ManagerId)
	if err != nil {
		return nil, err
	}
	return provider.(*SCloudprovider), nil
}

func (self *SExternalProject) GetCloudaccount() (*SCloudaccount, error) {
	provider, err := self.GetCloudprovider()
	if err != nil {
		return nil, err
	}
	return provider.GetCloudaccount()
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

	oldTenant, _ := db.TenantCacheManager.FetchTenantByIdOrNameInDomain(ctx, self.ProjectId, self.DomainId)
	oldDomain, oldProject := "", ""
	if oldTenant != nil {
		oldDomain, oldProject = oldTenant.Domain, oldTenant.Name
	}

	notes := struct {
		OldProjectId string
		OldProject   string
		OldDomainId  string
		OldDomain    string
		NewProjectId string
		NewProject   string
		NewDomainId  string
		NewDomain    string
	}{
		OldProjectId: self.ProjectId,
		OldProject:   oldProject,
		OldDomainId:  self.DomainId,
		OldDomain:    oldDomain,
		NewProjectId: tenant.Id,
		NewProject:   tenant.Name,
		NewDomainId:  tenant.DomainId,
		NewDomain:    tenant.Domain,
	}

	_, err = db.Update(self, func() error {
		self.ProjectId = tenant.Id
		self.DomainId = tenant.DomainId
		self.ProjectSrc = string(apis.OWNER_SOURCE_LOCAL)
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

	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, err
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
	q := manager.Query().IsNullOrEmpty("manager_id")
	projects := []SExternalProject{}
	err := db.FetchModelObjects(manager, q, &projects)
	if err != nil {
		return err
	}
	for i := range projects {
		if len(projects[i].CloudaccountId) > 0 {
			accountObj, err := CloudaccountManager.FetchById(projects[i].CloudaccountId)
			if err != nil {
				continue
			}
			account := accountObj.(*SCloudaccount)
			providers := account.GetCloudproviders()
			if len(providers) == 1 {
				_, err = db.Update(&projects[i], func() error {
					projects[i].ManagerId = providers[0].Id
					return nil
				})
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

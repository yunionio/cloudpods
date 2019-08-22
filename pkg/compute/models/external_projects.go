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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/compare"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/sqlchemy"
)

type SExternalProjectManager struct {
	db.SStandaloneResourceBaseManager

	db.SProjectizedResourceBaseManager
}

var ExternalProjectManager *SExternalProjectManager

func init() {
	ExternalProjectManager = &SExternalProjectManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SExternalProject{},
			"externalprojects_tbl",
			"externalproject",
			"externalprojects",
		),
	}
	ExternalProjectManager.SetVirtualObject(ExternalProjectManager)
}

type SExternalProject struct {
	db.SStandaloneResourceBase
	SManagedResourceBase

	db.SProjectizedResourceBase

	db.SExternalizedResourceBase
}

func (manager *SExternalProjectManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, manager)
}

func (self *SExternalProject) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return false
}

func (manager *SExternalProjectManager) getProjectsByProviderId(providerId string) ([]SExternalProject, error) {
	projects := []SExternalProject{}
	err := fetchByManagerId(manager, providerId, &projects)
	if err != nil {
		return nil, err
	}
	return projects, nil
}

func (self *SExternalProject) getCloudProviderInfo() SCloudProviderInfo {
	provider := self.GetCloudprovider()
	return MakeCloudProviderInfo(nil, nil, provider)
}

func (self *SExternalProject) getMoreDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	info := self.getCloudProviderInfo()

	extra.Update(jsonutils.Marshal(&info))

	tenant, err := db.TenantCacheManager.FetchTenantById(ctx, self.ProjectId)
	if err == nil {
		extra.Add(jsonutils.NewString(tenant.GetName()), "tenant")
	}
	return extra
}

func (self *SExternalProject) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreDetails(ctx, userCred, query, extra)
}

func (self *SExternalProject) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return self.getMoreDetails(ctx, userCred, query, extra), nil
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

func (manager *SExternalProjectManager) SyncProjects(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, projects []cloudprovider.ICloudProject) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	syncResult := compare.SyncResult{}

	dbProjects, err := manager.getProjectsByProviderId(provider.Id)
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
		err = commondb[i].SyncWithCloudProject(ctx, userCred, provider, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		_, err := manager.newFromCloudProject(ctx, userCred, provider, added[i])
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

func (self *SExternalProject) SyncWithCloudProject(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudProject) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Name = ext.GetName()
		self.IsEmulated = ext.IsEmulated()
		return nil
	})
	if err != nil {
		log.Errorf("SyncWithCloudProject fail %s", err)
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (manager *SExternalProjectManager) newFromCloudProject(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extProject cloudprovider.ICloudProject) (*SExternalProject, error) {
	project := SExternalProject{}
	project.SetModelManager(manager, &project)

	newName, err := db.GenerateName(manager, userCred, extProject.GetName())
	if err != nil {
		return nil, err
	}
	project.Name = newName
	project.ExternalId = extProject.GetGlobalId()
	project.IsEmulated = extProject.IsEmulated()
	project.ManagerId = provider.Id
	project.DomainId = provider.DomainId
	project.ProjectId = provider.ProjectId

	err = manager.TableSpec().Insert(&project)
	if err != nil {
		log.Errorf("newFromCloudProject fail %s", err)
		return nil, err
	}

	db.OpsLog.LogEvent(&project, db.ACT_CREATE, project.GetShortDesc(ctx), userCred)
	return &project, nil
}

func (self *SExternalProject) AllowPerformChangeProject(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "change-project")
}

func (self *SExternalProject) PerformChangeProject(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	project, err := data.GetString("project")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("project")
	}

	tenant, err := db.TenantCacheManager.FetchTenantByIdOrName(ctx, project)
	if err != nil {
		return nil, httperrors.NewNotFoundError("project %s not found", project)
	}

	if self.ProjectId == tenant.Id {
		return nil, nil
	}

	if self.DomainId != tenant.DomainId {
		return nil, httperrors.NewForbiddenError("not allow change project across domain")
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
		return nil
	})
	if err != nil {
		log.Errorf("Update external project error: %v", err)
		return nil, httperrors.NewGeneralError(err)
	}

	logclient.AddSimpleActionLog(self, logclient.ACT_CHANGE_OWNER, notes, userCred, true)
	return nil, nil
}

func (manager *SExternalProjectManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	var err error
	q, err = managedResourceFilterByAccount(q, query, "", nil)
	if err != nil {
		return nil, err
	}
	q = managedResourceFilterByCloudType(q, query, "", nil)

	q, err = manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	return q, nil
}

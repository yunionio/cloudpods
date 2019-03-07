package models

import (
	"context"
	"database/sql"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/compare"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SExternalProjectManager struct {
	db.SStandaloneResourceBaseManager
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
}

type SExternalProject struct {
	db.SStandaloneResourceBase
	SManagedResourceBase

	ProjectId string `name:"tenant_id" width:"128" charset:"ascii" nullable:"true" list:"admin" update:"admin"`
}

func (manager *SExternalProjectManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, manager)
}

func (self *SExternalProject) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SExternalProject) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if project := jsonutils.GetAnyString(data, []string{"project_id", "project", "tenant_id", "tenant"}); len(project) > 0 {
		_project, err := db.TenantCacheManager.FetchByIdOrName(userCred, project)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewTenantNotFoundError("tenant %s not find", project)
			}
			return nil, err
		}
		data.Set("tenant_id", jsonutils.NewString(_project.GetId()))
	}
	return self.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (manager *SExternalProjectManager) getProjectsByProvider(provider *SCloudprovider) ([]SExternalProject, error) {
	projects := []SExternalProject{}
	q := manager.Query()
	if provider != nil {
		q = q.Equals("manager_id", provider.Id)
	}
	err := db.FetchModelObjects(manager, q, &projects)
	if err != nil {
		return nil, err
	}
	return projects, nil
}

func (self *SExternalProject) getMoreDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
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
	project.SetModelManager(manager)
	q := manager.Query().Equals("external_id", externalId).Equals("manager_id", providerId)
	count := q.Count()
	if count == 0 {
		return nil, fmt.Errorf("no external project record %s for provider %s", externalId, providerId)
	}
	if count > 1 {
		return nil, fmt.Errorf("dumplicate external project record %s for provider %s", externalId, providerId)
	}
	return project, q.First(project)
}

func (manager *SExternalProjectManager) SyncProjects(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, projects []cloudprovider.ICloudProject) compare.SyncResult {
	lockman.LockClass(ctx, manager, manager.GetOwnerId(userCred))
	defer lockman.ReleaseClass(ctx, manager, manager.GetOwnerId(userCred))

	syncResult := compare.SyncResult{}

	dbProjects, err := manager.getProjectsByProvider(provider)
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
	project.SetModelManager(manager)

	project.Name = extProject.GetName()
	project.ExternalId = extProject.GetGlobalId()
	project.IsEmulated = extProject.IsEmulated()
	project.ManagerId = provider.Id
	project.ProjectId = provider.ProjectId

	err := manager.TableSpec().Insert(&project)
	if err != nil {
		log.Errorf("newFromCloudProject fail %s", err)
		return nil, err
	}

	db.OpsLog.LogEvent(&project, db.ACT_CREATE, project.GetShortDesc(ctx), userCred)
	return &project, nil
}

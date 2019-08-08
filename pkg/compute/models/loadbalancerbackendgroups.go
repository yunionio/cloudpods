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
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SLoadbalancerBackendGroupManager struct {
	SLoadbalancerLogSkipper
	db.SVirtualResourceBaseManager
}

var LoadbalancerBackendGroupManager *SLoadbalancerBackendGroupManager

func init() {
	LoadbalancerBackendGroupManager = &SLoadbalancerBackendGroupManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SLoadbalancerBackendGroup{},
			"loadbalancerbackendgroups_tbl",
			"loadbalancerbackendgroup",
			"loadbalancerbackendgroups",
		),
	}
}

type SLoadbalancerBackendGroup struct {
	db.SVirtualResourceBase
	SManagedResourceBase
	SCloudregionResourceBase

	Type           string `width:"36" charset:"ascii" nullable:"false" list:"user" default:"normal" create:"optional"`
	LoadbalancerId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
}

func (man *SLoadbalancerBackendGroupManager) pendingDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential, q *sqlchemy.SQuery) {
	lbbgs := []SLoadbalancerBackendGroup{}
	db.FetchModelObjects(man, q, &lbbgs)
	for _, lbbg := range lbbgs {
		lbbg.LBPendingDelete(ctx, userCred)
	}
}

func (man *SLoadbalancerBackendGroupManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	userProjId := userCred.GetProjectId()
	data := query.(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "loadbalancer", ModelKeyword: "loadbalancer", ProjectId: userProjId},
		{Key: "cloudregion", ModelKeyword: "cloudregion", ProjectId: userProjId},
		{Key: "manager", ModelKeyword: "cloudprovider", ProjectId: userProjId},
	})
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (man *SLoadbalancerBackendGroupManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	lbV := validators.NewModelIdOrNameValidator("loadbalancer", "loadbalancer", ownerProjId)
	err := lbV.Validate(data)
	if err != nil {
		return nil, err
	}
	if _, err := man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data); err != nil {
		return nil, err
	}
	lb := lbV.Model.(*SLoadbalancer)
	data.Set("manager_id", jsonutils.NewString(lb.ManagerId))
	data.Set("cloudregion_id", jsonutils.NewString(lb.CloudregionId))
	backends := []cloudprovider.SLoadbalancerBackend{}
	if data.Contains("backends") {
		if err := data.Unmarshal(&backends, "backends"); err != nil {
			return nil, err
		}
		for i := 0; i < len(backends); i++ {
			if len(backends[i].BackendType) == 0 {
				backends[i].BackendType = api.LB_BACKEND_GUEST
			}
			if backends[i].Weight < 0 || backends[i].Weight > 256 {
				return nil, httperrors.NewInputParameterError("weight %s not support, only support range 0 ~ 256")
			}
			if backends[i].Port < 1 || backends[i].Port > 65535 {
				return nil, httperrors.NewInputParameterError("port %s not support, only support range 1 ~ 65535")
			}
			if len(backends[i].ID) == 0 {
				return nil, httperrors.NewMissingParameterError("Missing backend id")
			}

			switch backends[i].BackendType {
			case api.LB_BACKEND_GUEST:
				_guest, err := GuestManager.FetchByIdOrName(userCred, backends[i].ID)
				if err != nil {
					if err == sql.ErrNoRows {
						return nil, httperrors.NewResourceNotFoundError("failed to find guest %s", backends[i].ID)
					}
					return nil, httperrors.NewGeneralError(err)
				}
				guest := _guest.(*SGuest)
				host := guest.GetHost()
				if host == nil {
					return nil, fmt.Errorf("error getting host of guest %s", guest.Name)
				}
				backends[i].ZoneId = host.ZoneId
				backends[i].HostName = host.Name
				backends[i].ID = guest.Id
				backends[i].Name = guest.Name
				backends[i].ExternalID = guest.ExternalId

				address, err := LoadbalancerBackendManager.GetGuestAddress(guest)
				if err != nil {
					return nil, err
				}
				backends[i].Address = address
			case api.LB_BACKEND_HOST:
				if !db.IsAdminAllowCreate(userCred, man) {
					return nil, httperrors.NewForbiddenError("only sysadmin can specify host as backend")
				}
				_host, err := HostManager.FetchByIdOrName(userCred, backends[i].ID)
				if err != nil {
					if err == sql.ErrNoRows {
						return nil, httperrors.NewResourceNotFoundError("failed to find host %s", backends[i].ID)
					}
					return nil, httperrors.NewGeneralError(err)
				}
				host := _host.(*SHost)
				backends[i].ID = host.Id
				backends[i].Name = host.Name
				backends[i].ExternalID = host.ExternalId
				backends[i].Address = host.AccessIp
			default:
				return nil, httperrors.NewInputParameterError("unexpected backend type %s", backends[i].BackendType)
			}
		}
	}
	data.Set("backends", jsonutils.Marshal(backends))
	region := lb.GetRegion()
	if region == nil {
		return nil, httperrors.NewResourceNotFoundError("failed to find region for loadbalancer %s", lb.Name)
	}
	return region.GetDriver().ValidateCreateLoadbalancerBackendGroupData(ctx, userCred, data, lb, backends)
}

func (lbbg *SLoadbalancerBackendGroup) GetLoadbalancerListenerRules() ([]SLoadbalancerListenerRule, error) {
	q := LoadbalancerListenerRuleManager.Query().Equals("backend_group_id", lbbg.Id).IsFalse("pending_deleted")
	rules := []SLoadbalancerListenerRule{}
	err := db.FetchModelObjects(LoadbalancerListenerRuleManager, q, &rules)
	if err != nil {
		return nil, err
	}
	return rules, nil
}

func (lbbg *SLoadbalancerBackendGroup) GetLoadbalancerListeners() ([]SLoadbalancerListener, error) {
	q := LoadbalancerListenerManager.Query().Equals("backend_group_id", lbbg.Id).IsFalse("pending_deleted")
	listeners := []SLoadbalancerListener{}
	err := db.FetchModelObjects(LoadbalancerListenerManager, q, &listeners)
	if err != nil {
		return nil, err
	}
	return listeners, nil
}

func (lbbg *SLoadbalancerBackendGroup) GetLoadbalancer() *SLoadbalancer {
	lb, err := LoadbalancerManager.FetchById(lbbg.LoadbalancerId)
	if err != nil {
		log.Errorf("failed to find loadbalancer for backendgroup %s", lbbg.Name)
		return nil
	}
	return lb.(*SLoadbalancer)
}

func (llbg *SLoadbalancerBackendGroup) GetRegion() *SCloudregion {
	if loadbalancer := llbg.GetLoadbalancer(); loadbalancer != nil {
		return loadbalancer.GetRegion()
	}
	return nil
}

func (lbbg *SLoadbalancerBackendGroup) GetIRegion() (cloudprovider.ICloudRegion, error) {
	if loadbalancer := lbbg.GetLoadbalancer(); loadbalancer != nil {
		return loadbalancer.GetIRegion()
	}
	return nil, fmt.Errorf("failed to find loadbalancer for backendgroup %s", lbbg.Name)
}

func (lbbg *SLoadbalancerBackendGroup) GetBackends() ([]SLoadbalancerBackend, error) {
	backends := make([]SLoadbalancerBackend, 0)
	q := LoadbalancerBackendManager.Query().Equals("backend_group_id", lbbg.GetId()).IsFalse("pending_deleted")
	err := db.FetchModelObjects(LoadbalancerBackendManager, q, &backends)
	if err != nil {
		return nil, err
	}
	return backends, nil
}

// 返回值 TotalRef
func (lbbg *SLoadbalancerBackendGroup) RefCount() (int, error) {
	men := lbbg.getRefManagers()
	var count int
	for _, m := range men {
		cnt, err := lbbg.refCount(m)
		if err != nil {
			return -1, err
		}
		count += cnt
	}

	return count, nil
}

func (lbbg *SLoadbalancerBackendGroup) refCount(men db.IModelManager) (int, error) {
	t := men.TableSpec().Instance()
	pdF := t.Field("pending_deleted")
	return t.Query().
		Equals("backend_group_id", lbbg.Id).
		Filter(sqlchemy.OR(sqlchemy.IsNull(pdF), sqlchemy.IsFalse(pdF))).
		CountWithError()
}

func (lbbg *SLoadbalancerBackendGroup) getRefManagers() []db.IModelManager {
	// 引用Backend Group的数据库
	return []db.IModelManager{
		LoadbalancerManager,
		LoadbalancerListenerManager,
		LoadbalancerListenerRuleManager,
	}

}

func (lbbg *SLoadbalancerBackendGroup) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (lbbg *SLoadbalancerBackendGroup) ValidateDeleteCondition(ctx context.Context) error {
	men := lbbg.getRefManagers()
	for _, m := range men {
		n, err := lbbg.refCount(m)
		if err != nil {
			return httperrors.NewInternalServerError("get refCount fail %s", err.Error())
		}
		if n > 0 {
			return httperrors.NewResourceBusyError("backend group %s is still referred to by %d %s",
				lbbg.Id, n, m.KeywordPlural())
		}
	}

	return nil
}

func (lbbg *SLoadbalancerBackendGroup) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := lbbg.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	{
		lb, err := LoadbalancerManager.FetchById(lbbg.LoadbalancerId)
		if err != nil {
			log.Errorf("loadbalancer backend group %s(%s): fetch loadbalancer (%s) error: %s",
				lbbg.Name, lbbg.Id, lbbg.LoadbalancerId, err)
			return extra
		}
		extra.Set("loadbalancer", jsonutils.NewString(lb.GetName()))
	}
	regionInfo := lbbg.SCloudregionResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if regionInfo != nil {
		extra.Update(regionInfo)
	}
	return extra
}

func (lbbg *SLoadbalancerBackendGroup) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra := lbbg.GetCustomizeColumns(ctx, userCred, query)
	return extra, nil
}

func (lbbg *SLoadbalancerBackendGroup) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbbg.SVirtualResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)
	params := jsonutils.NewDict()
	backends, _ := data.Get("backends")
	if backends != nil {
		params.Add(backends, "backends")
	}
	lbbg.SetStatus(userCred, api.LB_CREATING, "")
	if err := lbbg.StartLoadBalancerBackendGroupCreateTask(ctx, userCred, params, ""); err != nil {
		log.Errorf("Failed to create loadbalancer backendgroup error: %v", err)
	}
}

func (lbbg *SLoadbalancerBackendGroup) StartLoadBalancerBackendGroupCreateTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerLoadbalancerBackendGroupCreateTask", lbbg, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbbg *SLoadbalancerBackendGroup) LBPendingDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	lbbg.pendingDeleteSubs(ctx, userCred)
	lbbg.DoPendingDelete(ctx, userCred)
}

func (lbbg *SLoadbalancerBackendGroup) pendingDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential) {
	lbbg.DoPendingDelete(ctx, userCred)
	subMan := LoadbalancerBackendManager
	ownerProjId := lbbg.GetOwnerProjectId()

	lockman.LockClass(ctx, subMan, ownerProjId)
	defer lockman.ReleaseClass(ctx, subMan, ownerProjId)
	q := subMan.Query().Equals("backend_group_id", lbbg.Id)
	subMan.pendingDeleteSubs(ctx, userCred, q)
}

func (lbbg *SLoadbalancerBackendGroup) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, lbbg, "purge")
}

func (lbbg *SLoadbalancerBackendGroup) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	parasm := jsonutils.NewDict()
	parasm.Add(jsonutils.JSONTrue, "purge")
	return nil, lbbg.StartLoadBalancerBackendGroupDeleteTask(ctx, userCred, parasm, "")
}

func (lbbg *SLoadbalancerBackendGroup) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	lbbg.SetStatus(userCred, api.LB_STATUS_DELETING, "")
	return lbbg.StartLoadBalancerBackendGroupDeleteTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (lbbg *SLoadbalancerBackendGroup) StartLoadBalancerBackendGroupDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerBackendGroupDeleteTask", lbbg, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbbg *SLoadbalancerBackendGroup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (man *SLoadbalancerBackendGroupManager) getLoadbalancerBackendgroupsByLoadbalancer(lb *SLoadbalancer) ([]SLoadbalancerBackendGroup, error) {
	lbbgs := []SLoadbalancerBackendGroup{}
	q := man.Query().Equals("loadbalancer_id", lb.Id).IsFalse("pending_deleted")
	if err := db.FetchModelObjects(man, q, &lbbgs); err != nil {
		log.Errorf("failed to get lbbgs for lb: %s error: %v", lb.Name, err)
		return nil, err
	}
	return lbbgs, nil
}

func (man *SLoadbalancerBackendGroupManager) SyncLoadbalancerBackendgroups(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, lb *SLoadbalancer, lbbgs []cloudprovider.ICloudLoadbalancerBackendGroup, syncRange *SSyncRange) ([]SLoadbalancerBackendGroup, []cloudprovider.ICloudLoadbalancerBackendGroup, compare.SyncResult) {
	syncOwnerId := provider.ProjectId

	lockman.LockClass(ctx, man, syncOwnerId)
	defer lockman.ReleaseClass(ctx, man, syncOwnerId)

	localLbgs := []SLoadbalancerBackendGroup{}
	remoteLbbgs := []cloudprovider.ICloudLoadbalancerBackendGroup{}
	syncResult := compare.SyncResult{}

	dbLbbgs, err := man.getLoadbalancerBackendgroupsByLoadbalancer(lb)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	removed := []SLoadbalancerBackendGroup{}
	commondb := []SLoadbalancerBackendGroup{}
	commonext := []cloudprovider.ICloudLoadbalancerBackendGroup{}
	added := []cloudprovider.ICloudLoadbalancerBackendGroup{}

	err = compare.CompareSets(dbLbbgs, lbbgs, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].syncRemoveCloudLoadbalancerBackendgroup(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudLoadbalancerBackendgroup(ctx, userCred, lb, commonext[i], provider.ProjectId)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			localLbgs = append(localLbgs, commondb[i])
			remoteLbbgs = append(remoteLbbgs, commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		new, err := man.newFromCloudLoadbalancerBackendgroup(ctx, userCred, lb, added[i], syncOwnerId)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, new, added[i])
			localLbgs = append(localLbgs, *new)
			remoteLbbgs = append(remoteLbbgs, added[i])
			syncResult.Add()
		}
	}
	return localLbgs, remoteLbbgs, syncResult
}

/*func (lbbg *SLoadbalancerBackendGroup) constructFieldsFromCloudBackendgroup(lb *SLoadbalancer, extLoadbalancerBackendgroup cloudprovider.ICloudLoadbalancerBackendGroup) {
	// 对于腾讯云,backend group 名字以本地为准
	if lbbg.GetProviderName() != CLOUD_PROVIDER_QCLOUD || len(lbbg.Name) == 0 {
		lbbg.Name = extLoadbalancerBackendgroup.GetName()
	}

	lbbg.Type = extLoadbalancerBackendgroup.GetType()
	lbbg.Status = extLoadbalancerBackendgroup.GetStatus()
}*/

func (lbbg *SLoadbalancerBackendGroup) syncRemoveCloudLoadbalancerBackendgroup(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbbg)
	defer lockman.ReleaseObject(ctx, lbbg)

	err := lbbg.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		err = lbbg.SetStatus(userCred, api.LB_STATUS_UNKNOWN, "sync to delete")
	} else {
		lbbg.LBPendingDelete(ctx, userCred)
	}
	return err
}

func (lbbg *SLoadbalancerBackendGroup) SyncWithCloudLoadbalancerBackendgroup(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, extLoadbalancerBackendgroup cloudprovider.ICloudLoadbalancerBackendGroup, projectId string) error {
	diff, err := db.UpdateWithLock(ctx, lbbg, func() error {
		lbbg.Type = extLoadbalancerBackendgroup.GetType()
		lbbg.Status = extLoadbalancerBackendgroup.GetStatus()
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(lbbg, diff, userCred)

	SyncCloudProject(userCred, lbbg, projectId, extLoadbalancerBackendgroup, lb.ManagerId)

	if extLoadbalancerBackendgroup.IsDefault() {
		diff, err := db.UpdateWithLock(ctx, lb, func() error {
			lb.BackendGroupId = lbbg.Id
			return nil
		})
		if err != nil {
			log.Errorf("failed to set backendgroup id for lb %s error: %v", lb.Name, err)
			return err
		}
		db.OpsLog.LogEvent(lb, db.ACT_UPDATE, diff, userCred)
	}

	return err
}

func (man *SLoadbalancerBackendGroupManager) newFromCloudLoadbalancerBackendgroup(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, extLoadbalancerBackendgroup cloudprovider.ICloudLoadbalancerBackendGroup, projectId string) (*SLoadbalancerBackendGroup, error) {
	lbbg := &SLoadbalancerBackendGroup{}
	lbbg.SetModelManager(man)

	lbbg.LoadbalancerId = lb.Id
	lbbg.ExternalId = extLoadbalancerBackendgroup.GetGlobalId()

	lbbg.CloudregionId = lb.CloudregionId
	lbbg.ManagerId = lb.ManagerId

	/*lbbg.constructFieldsFromCloudBackendgroup(lb, extLoadbalancerBackendgroup)
	if lbbg.GetProviderName() != CLOUD_PROVIDER_QCLOUD || len(lbbg.Name) == 0 {

	}*/

	newName, err := db.GenerateName(man, projectId, extLoadbalancerBackendgroup.GetName())
	if err != nil {
		return nil, err
	}

	lbbg.Name = newName

	lbbg.Type = extLoadbalancerBackendgroup.GetType()
	lbbg.Status = extLoadbalancerBackendgroup.GetStatus()

	err = man.TableSpec().Insert(lbbg)
	if err != nil {
		return nil, err
	}

	SyncCloudProject(userCred, lbbg, projectId, extLoadbalancerBackendgroup, lb.ManagerId)

	db.OpsLog.LogEvent(lbbg, db.ACT_CREATE, lbbg.GetShortDesc(ctx), userCred)

	if extLoadbalancerBackendgroup.IsDefault() {
		_, err := db.Update(lb, func() error {
			lb.BackendGroupId = lbbg.Id
			return nil
		})
		if err != nil {
			log.Errorf("failed to set backendgroup id for lb %s error: %v", lb.Name, err)
		}
	}
	return lbbg, nil
}

func (man *SLoadbalancerBackendGroupManager) initBackendGroupType() error {
	backendgroups := []SLoadbalancerBackendGroup{}
	q := man.Query()
	q = q.Filter(sqlchemy.IsNullOrEmpty(q.Field("type")))
	if err := db.FetchModelObjects(man, q, &backendgroups); err != nil {
		log.Errorf("failed fetching backendgroups with empty type error: %v", err)
		return err
	}
	for i := 0; i < len(backendgroups); i++ {
		_, err := db.Update(&backendgroups[i], func() error {
			backendgroups[i].Type = api.LB_BACKENDGROUP_TYPE_NORMAL
			return nil
		})
		if err != nil {
			log.Errorf("failed setting backendgroup %s(%s) type error: %v", backendgroups[i].Name, backendgroups[i].Id, err)
		}
	}
	return nil
}

func (man *SLoadbalancerBackendGroupManager) InitializeData() error {
	if err := man.initBackendGroupType(); err != nil {
		return err
	}
	return man.initBackendGroupRegion()
}

func (manager *SLoadbalancerBackendGroupManager) initBackendGroupRegion() error {
	groups := []SLoadbalancerBackendGroup{}
	q := manager.Query()
	q = q.Filter(sqlchemy.IsNullOrEmpty(q.Field("cloudregion_id")))
	if err := db.FetchModelObjects(manager, q, &groups); err != nil {
		return err
	}
	for i := 0; i < len(groups); i++ {
		group := &groups[i]
		if lb := group.GetLoadbalancer(); lb != nil && len(lb.CloudregionId) > 0 {
			_, err := db.Update(group, func() error {
				group.CloudregionId = lb.CloudregionId
				group.ManagerId = lb.ManagerId
				return nil
			})
			if err != nil {
				log.Errorf("failed to update loadbalancer backendgroup %s cloudregion_id", group.Name)
			}
		}
	}
	return nil
}

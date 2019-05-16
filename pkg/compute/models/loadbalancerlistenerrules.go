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

type SLoadbalancerListenerRuleManager struct {
	SLoadbalancerLogSkipper
	db.SVirtualResourceBaseManager
}

var LoadbalancerListenerRuleManager *SLoadbalancerListenerRuleManager

func init() {
	LoadbalancerListenerRuleManager = &SLoadbalancerListenerRuleManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SLoadbalancerListenerRule{},
			"loadbalancerlistenerrules_tbl",
			"loadbalancerlistenerrule",
			"loadbalancerlistenerrules",
		),
	}
	LoadbalancerListenerRuleManager.SetVirtualObject(LoadbalancerListenerRuleManager)
}

type SLoadbalancerListenerRule struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase

	ListenerId     string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	BackendGroupId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user"`

	Domain string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	Path   string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"optional"`

	SLoadbalancerHealthCheck // 目前只有腾讯云HTTP、HTTPS类型的健康检查是和规则绑定的。
	SLoadbalancerHTTPRateLimiter
}

func loadbalancerListenerRuleCheckUniqueness(ctx context.Context, lbls *SLoadbalancerListener, domain, path string) error {
	q := LoadbalancerListenerRuleManager.Query().
		IsFalse("pending_deleted").
		Equals("listener_id", lbls.Id).
		Equals("domain", domain).
		Equals("path", path)
	var lblsr SLoadbalancerListenerRule
	q.First(&lblsr)
	if len(lblsr.Id) > 0 {
		return httperrors.NewConflictError("rule %s/%s already occupied by rule %s(%s)", domain, path, lblsr.Name, lblsr.Id)
	}
	return nil
}

func (man *SLoadbalancerListenerRuleManager) pendingDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential, q *sqlchemy.SQuery) {
	subs := []SLoadbalancerListenerRule{}
	db.FetchModelObjects(man, q, &subs)
	for _, sub := range subs {
		sub.DoPendingDelete(ctx, userCred)
	}
}

func (man *SLoadbalancerListenerRuleManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	// userProjId := userCred.GetProjectId()
	data := query.(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "listener", ModelKeyword: "loadbalancerlistener", OwnerId: userCred},
		{Key: "backend_group", ModelKeyword: "loadbalancerbackendgroup", OwnerId: userCred},
	})
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (man *SLoadbalancerListenerRuleManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	listenerV := validators.NewModelIdOrNameValidator("listener", "loadbalancerlistener", ownerId)
	backendGroupV := validators.NewModelIdOrNameValidator("backend_group", "loadbalancerbackendgroup", ownerId)
	domainV := validators.NewDomainNameValidator("domain")
	pathV := validators.NewURLPathValidator("path")
	keyV := map[string]validators.IValidator{
		"status": validators.NewStringChoicesValidator("status", api.LB_STATUS_SPEC).Default(api.LB_STATUS_ENABLED),

		"listener":      listenerV,
		"backend_group": backendGroupV,
		"domain":        domainV.AllowEmpty(true).Default(""),
		"path":          pathV.Default(""),

		"http_request_rate":         validators.NewNonNegativeValidator("http_request_rate").Default(0),
		"http_request_rate_per_src": validators.NewNonNegativeValidator("http_request_rate_per_src").Default(0),
	}
	for _, v := range keyV {
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}
	listener := listenerV.Model.(*SLoadbalancerListener)
	data.Set("cloudregion_id", jsonutils.NewString(listener.CloudregionId))
	data.Set("manager_id", jsonutils.NewString(listener.ManagerId))
	listenerType := listener.ListenerType
	if listenerType != api.LB_LISTENER_TYPE_HTTP && listenerType != api.LB_LISTENER_TYPE_HTTPS {
		return nil, httperrors.NewInputParameterError("listener type must be http/https, got %s", listenerType)
	}
	{
		if lbbg, ok := backendGroupV.Model.(*SLoadbalancerBackendGroup); ok && lbbg.LoadbalancerId != listener.LoadbalancerId {
			return nil, httperrors.NewInputParameterError("backend group %s(%s) belongs to loadbalancer %s instead of %s",
				lbbg.Name, lbbg.Id, lbbg.LoadbalancerId, listener.LoadbalancerId)
		} else {
			// 腾讯云backend group只能1v1关联
			if listener.GetProviderName() == api.CLOUD_PROVIDER_QCLOUD {
				count, err := lbbg.RefCount()
				if err != nil {
					return nil, httperrors.NewInternalServerError("get lbbg RefCount fail %s", err)
				}
				if count > 0 {
					return nil, httperrors.NewResourceBusyError("backendgroup already related with other listener/rule")
				}
			}
		}
	}
	err := loadbalancerListenerRuleCheckUniqueness(ctx, listener, domainV.Value, pathV.Value)
	if err != nil {
		return nil, err
	}
	if _, err := man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data); err != nil {
		return nil, err
	}
	region := listener.GetRegion()
	if region == nil {
		return nil, httperrors.NewResourceNotFoundError("failed to find region for loadbalancer listener %s", listener.Name)
	}

	return region.GetDriver().ValidateCreateLoadbalancerListenerRuleData(ctx, userCred, data, backendGroupV.Model)
}

func (lbr *SLoadbalancerListenerRule) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbr.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	lbr.SetStatus(userCred, api.LB_CREATING, "")
	if err := lbr.StartLoadBalancerListenerRuleCreateTask(ctx, userCred, ""); err != nil {
		log.Errorf("Failed to create loadbalancer listener rule error: %v", err)
	}
}

func (lbr *SLoadbalancerListenerRule) StartLoadBalancerListenerRuleCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerListenerRuleCreateTask", lbr, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbr *SLoadbalancerListenerRule) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, lbr, "purge")
}

func (lbr *SLoadbalancerListenerRule) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	parasm := jsonutils.NewDict()
	parasm.Add(jsonutils.JSONTrue, "purge")
	return nil, lbr.StartLoadBalancerListenerRuleDeleteTask(ctx, userCred, parasm, "")
}

func (lbr *SLoadbalancerListenerRule) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	lbr.SetStatus(userCred, api.LB_STATUS_DELETING, "")
	return lbr.StartLoadBalancerListenerRuleDeleteTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (lbr *SLoadbalancerListenerRule) StartLoadBalancerListenerRuleDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerListenerRuleDeleteTask", lbr, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbr *SLoadbalancerListenerRule) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return lbr.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, lbr, "status")
}

func (lbr *SLoadbalancerListenerRule) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	backendGroupV := validators.NewModelIdOrNameValidator("backend_group", "loadbalancerbackendgroup", lbr.GetOwnerId())
	keyV := map[string]validators.IValidator{
		"backend_group":             backendGroupV,
		"http_request_rate":         validators.NewNonNegativeValidator("http_request_rate"),
		"http_request_rate_per_src": validators.NewNonNegativeValidator("http_request_rate_per_src"),
	}
	for _, v := range keyV {
		v.Optional(true)
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}
	if backendGroup, ok := backendGroupV.Model.(*SLoadbalancerBackendGroup); ok && backendGroup.Id != lbr.BackendGroupId {
		listenerM, err := LoadbalancerListenerManager.FetchById(lbr.ListenerId)
		if err != nil {
			return nil, httperrors.NewInputParameterError("loadbalancerlistenerrule %s(%s): fetching listener %s failed",
				lbr.Name, lbr.Id, lbr.ListenerId)
		}
		listener := listenerM.(*SLoadbalancerListener)
		if backendGroup.LoadbalancerId != listener.LoadbalancerId {
			return nil, httperrors.NewInputParameterError("backend group %s(%s) belongs to loadbalancer %s instead of %s",
				backendGroup.Name, backendGroup.Id, backendGroup.LoadbalancerId, listener.LoadbalancerId)
		}
	}
	return lbr.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (lbr *SLoadbalancerListenerRule) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := lbr.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if lbr.BackendGroupId == "" {
		log.Errorf("loadbalancer listener rule %s(%s): empty backend group field", lbr.Name, lbr.Id)
		return extra
	}
	lbbg, err := LoadbalancerBackendGroupManager.FetchById(lbr.BackendGroupId)
	if err != nil {
		log.Errorf("loadbalancer listener rule %s(%s): fetch backend group (%s) error: %s",
			lbr.Name, lbr.Id, lbr.BackendGroupId, err)
		return extra
	}
	extra.Set("backend_group", jsonutils.NewString(lbbg.GetName()))

	regionInfo := lbr.SCloudregionResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if regionInfo != nil {
		extra.Update(regionInfo)
	}
	return extra
}

func (lbr *SLoadbalancerListenerRule) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra := lbr.GetCustomizeColumns(ctx, userCred, query)
	return extra, nil
}

func (lbr *SLoadbalancerListenerRule) GetLoadbalancerListener() *SLoadbalancerListener {
	listener, err := LoadbalancerListenerManager.FetchById(lbr.ListenerId)
	if err != nil {
		log.Errorf("failed to find listener for loadbalancer listener rule %s", lbr.Name)
		return nil
	}
	return listener.(*SLoadbalancerListener)
}

func (lbr *SLoadbalancerListenerRule) GetRegion() *SCloudregion {
	if listener := lbr.GetLoadbalancerListener(); listener != nil {
		return listener.GetRegion()
	}
	return nil
}

func (lbr *SLoadbalancerListenerRule) GetLoadbalancerBackendGroup() *SLoadbalancerBackendGroup {
	group, err := LoadbalancerBackendGroupManager.FetchById(lbr.BackendGroupId)
	if err != nil {
		return nil
	}
	return group.(*SLoadbalancerBackendGroup)
}

func (lbr *SLoadbalancerListenerRule) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

// Delete, Update

func (man *SLoadbalancerListenerRuleManager) getLoadbalancerListenerRulesByListener(listener *SLoadbalancerListener) ([]SLoadbalancerListenerRule, error) {
	rules := []SLoadbalancerListenerRule{}
	q := man.Query().Equals("listener_id", listener.Id).IsFalse("pending_deleted")
	if err := db.FetchModelObjects(man, q, &rules); err != nil {
		log.Errorf("failed to get lb listener rules for listener %s error: %v", listener.Name, err)
		return nil, err
	}
	return rules, nil
}

func (man *SLoadbalancerListenerRuleManager) SyncLoadbalancerListenerRules(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, listener *SLoadbalancerListener, rules []cloudprovider.ICloudLoadbalancerListenerRule, syncRange *SSyncRange) compare.SyncResult {
	syncOwnerId := provider.GetOwnerId()

	lockman.LockClass(ctx, man, db.GetLockClassKey(man, syncOwnerId))
	defer lockman.ReleaseClass(ctx, man, db.GetLockClassKey(man, syncOwnerId))

	syncResult := compare.SyncResult{}

	dbRules, err := man.getLoadbalancerListenerRulesByListener(listener)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := []SLoadbalancerListenerRule{}
	commondb := []SLoadbalancerListenerRule{}
	commonext := []cloudprovider.ICloudLoadbalancerListenerRule{}
	added := []cloudprovider.ICloudLoadbalancerListenerRule{}

	err = compare.CompareSets(dbRules, rules, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].syncRemoveCloudLoadbalancerListenerRule(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudLoadbalancerListenerRule(ctx, userCred, commonext[i], syncOwnerId)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		local, err := man.newFromCloudLoadbalancerListenerRule(ctx, userCred, listener, added[i], syncOwnerId)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, local, added[i])
			syncResult.Add()
		}
	}

	return syncResult
}

func (lbr *SLoadbalancerListenerRule) constructFieldsFromCloudListenerRule(userCred mcclient.TokenCredential, extRule cloudprovider.ICloudLoadbalancerListenerRule) {
	// lbr.Name = extRule.GetName()
	lbr.Domain = extRule.GetDomain()
	lbr.Path = extRule.GetPath()
	lbr.Status = extRule.GetStatus()
	if groupId := extRule.GetBackendGroupId(); len(groupId) > 0 {
		// 腾讯云兼容代码。主要目的是在关联listener rule时回写一个fake的backend group external id
		if lbr.GetProviderName() == api.CLOUD_PROVIDER_QCLOUD && len(groupId) > 0 && len(lbr.BackendGroupId) > 0 {
			ilbbg, err := LoadbalancerBackendGroupManager.FetchById(lbr.BackendGroupId)
			lbbg := ilbbg.(*SLoadbalancerBackendGroup)
			if err == nil && (len(lbbg.ExternalId) == 0 || lbbg.ExternalId != groupId) {
				err = db.SetExternalId(lbbg, userCred, groupId)
				if err != nil {
					log.Errorf("Update loadbalancer BackendGroup(%s) external id failed: %s", lbbg.GetId(), err)
				}
			}
		}

		if lbr.GetProviderName() == api.CLOUD_PROVIDER_HUAWEI {
			group, err := db.FetchByExternalId(HuaweiCachedLbbgManager, groupId)
			if err != nil {
				log.Errorf("Fetch huawei loadbalancer backendgroup by external id %s failed: %s", groupId, err)
			}

			lbr.BackendGroupId = group.(*SHuaweiCachedLbbg).BackendGroupId
		} else if backendgroup, err := db.FetchByExternalId(LoadbalancerBackendGroupManager, groupId); err == nil {
			lbr.BackendGroupId = backendgroup.GetId()
		}
	}
}

func (man *SLoadbalancerListenerRuleManager) newFromCloudLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, listener *SLoadbalancerListener, extRule cloudprovider.ICloudLoadbalancerListenerRule, syncOwnerId mcclient.IIdentityProvider) (*SLoadbalancerListenerRule, error) {
	lbr := &SLoadbalancerListenerRule{}
	lbr.SetModelManager(man, lbr)

	lbr.ExternalId = extRule.GetGlobalId()
	lbr.ListenerId = listener.Id
	lbr.ManagerId = listener.ManagerId
	lbr.CloudregionId = listener.CloudregionId

	newName, err := db.GenerateName(man, syncOwnerId, extRule.GetName())
	if err != nil {
		return nil, err
	}
	lbr.Name = newName
	lbr.constructFieldsFromCloudListenerRule(userCred, extRule)

	err = man.TableSpec().Insert(lbr)

	if err != nil {
		log.Errorf("newFromCloudLoadbalancerListenerRule fail %s", err)
		return nil, err
	}

	groupId := extRule.GetBackendGroupId()
	if lbr.GetProviderName() == api.CLOUD_PROVIDER_HUAWEI && len(groupId) > 0 {
		group, err := db.FetchByExternalId(HuaweiCachedLbbgManager, groupId)
		if err != nil {
			log.Errorf("Fetch huawei loadbalancer backendgroup by external id %s failed: %s", groupId, err)
		}

		cachedGroup := group.(*SHuaweiCachedLbbg)
		_, err = db.UpdateWithLock(context.Background(), cachedGroup, func() error {
			cachedGroup.AssociatedId = lbr.GetId()
			cachedGroup.AssociatedType = api.LB_ASSOCIATE_TYPE_RULE
			return nil
		})
		if err != nil {
			log.Errorf("Update huawei loadbalancer backendgroup cache %s failed: %s", groupId, err)
		}
	}

	SyncCloudProject(userCred, lbr, syncOwnerId, extRule, listener.ManagerId)

	db.OpsLog.LogEvent(lbr, db.ACT_CREATE, lbr.GetShortDesc(ctx), userCred)

	return lbr, nil
}

func (lbr *SLoadbalancerListenerRule) syncRemoveCloudLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbr)
	defer lockman.ReleaseObject(ctx, lbr)

	err := lbr.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		err = lbr.SetStatus(userCred, api.LB_STATUS_UNKNOWN, "sync to delete")
	} else {
		err = lbr.DoPendingDelete(ctx, userCred)
	}
	return err
}

func (lbr *SLoadbalancerListenerRule) SyncWithCloudLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, extRule cloudprovider.ICloudLoadbalancerListenerRule, syncOwnerId mcclient.IIdentityProvider) error {
	listener := lbr.GetLoadbalancerListener()
	diff, err := db.UpdateWithLock(ctx, lbr, func() error {
		lbr.constructFieldsFromCloudListenerRule(userCred, extRule)
		lbr.ManagerId = listener.ManagerId
		lbr.CloudregionId = listener.CloudregionId
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(lbr, diff, userCred)

	SyncCloudProject(userCred, lbr, syncOwnerId, extRule, listener.ManagerId)

	return nil
}

func (manager *SLoadbalancerListenerRuleManager) InitializeData() error {
	rules := []SLoadbalancerListenerRule{}
	q := manager.Query()
	q = q.Filter(sqlchemy.IsNullOrEmpty(q.Field("cloudregion_id")))
	if err := db.FetchModelObjects(manager, q, &rules); err != nil {
		return err
	}
	for i := 0; i < len(rules); i++ {
		rule := &rules[i]
		if listener := rule.GetLoadbalancerListener(); listener != nil && len(listener.CloudregionId) > 0 {
			_, err := db.Update(rule, func() error {
				rule.CloudregionId = listener.CloudregionId
				rule.ManagerId = listener.ManagerId
				return nil
			})
			if err != nil {
				log.Errorf("failed to update loadbalancer listener rule %s cloudregion_id", rule.Name)
			}
		}
	}
	return nil
}

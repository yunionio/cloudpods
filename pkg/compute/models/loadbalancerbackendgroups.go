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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=loadbalancerbackendgroup
// +onecloud:swagger-gen-model-plural=loadbalancerbackendgroups
type SLoadbalancerBackendGroupManager struct {
	SLoadbalancerLogSkipper
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SLoadbalancerResourceBaseManager
}

var LoadbalancerBackendGroupManager *SLoadbalancerBackendGroupManager

func init() {
	LoadbalancerBackendGroupManager = &SLoadbalancerBackendGroupManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SLoadbalancerBackendGroup{},
			"loadbalancerbackendgroups_tbl",
			"loadbalancerbackendgroup",
			"loadbalancerbackendgroups",
		),
	}
	LoadbalancerBackendGroupManager.SetVirtualObject(LoadbalancerBackendGroupManager)
}

type SLoadbalancerBackendGroup struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase

	SLoadbalancerResourceBase `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	Type string `width:"36" charset:"ascii" nullable:"false" list:"user" default:"normal" create:"optional"`
}

func (manager *SLoadbalancerBackendGroupManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeProject
}

func (self *SLoadbalancerBackendGroup) GetOwnerId() mcclient.IIdentityProvider {
	lb, err := self.GetLoadbalancer()
	if err != nil {
		return nil
	}
	return lb.GetOwnerId()
}

func (manager *SLoadbalancerBackendGroupManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	lbId, _ := data.GetString("loadbalancer_id")
	if len(lbId) > 0 {
		lb, err := db.FetchById(LoadbalancerManager, lbId)
		if err != nil {
			return nil, errors.Wrapf(err, "db.FetchById(LoadbalancerManager, %s)", lbId)
		}
		return lb.(*SLoadbalancer).GetOwnerId(), nil
	}
	return db.FetchProjectInfo(ctx, data)
}

func (manager *SLoadbalancerBackendGroupManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if ownerId != nil {
		sq := LoadbalancerManager.Query("id")
		switch scope {
		case rbacscope.ScopeProject:
			sq = sq.Equals("tenant_id", ownerId.GetProjectId())
			return q.In("loadbalancer_id", sq.SubQuery())
		case rbacscope.ScopeDomain:
			sq = sq.Equals("domain_id", ownerId.GetProjectDomainId())
			return q.In("loadbalancer_id", sq.SubQuery())
		}
	}
	return q
}

// 负载均衡后端服务器组列表
func (man *SLoadbalancerBackendGroupManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerBackendGroupListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemFilter")
	}

	q, err = man.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = man.SLoadbalancerResourceBaseManager.ListItemFilter(ctx, q, userCred, query.LoadbalancerFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLoadbalancerResourceBaseManager.ListItemFilter")
	}

	if query.NoRef != nil && *query.NoRef {
		q, err = man.FilterZeroRefBackendGroup(q)
		if err != nil {
			log.Errorf("SLoadbalancerBackendGroupManager ListItemFilter %s", err)
			return nil, httperrors.NewInternalServerError("query backend group releated resource failed.")
		}
	}

	if len(query.Type) > 0 {
		q = q.In("type", query.Type)
	}

	return q, nil
}

func (man *SLoadbalancerBackendGroupManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerBackendGroupListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SLoadbalancerResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.LoadbalancerFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLoadbalancerResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (man *SLoadbalancerBackendGroupManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SLoadbalancerResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

type sBackendgroup struct {
	Name           string
	LoadbalancerId string
}

func (self *SLoadbalancerBackendGroup) GetUniqValues() jsonutils.JSONObject {
	return jsonutils.Marshal(sBackendgroup{Name: self.Name, LoadbalancerId: self.LoadbalancerId})
}

func (manager *SLoadbalancerBackendGroupManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	info := sBackendgroup{}
	data.Unmarshal(&info)
	return jsonutils.Marshal(info)
}

func (manager *SLoadbalancerBackendGroupManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	info := sBackendgroup{}
	values.Unmarshal(&info)
	if len(info.LoadbalancerId) > 0 {
		q = q.Equals("loadbalancer_id", info.LoadbalancerId)
	}
	if len(info.Name) > 0 {
		q = q.Equals("name", info.Name)
	}
	return q
}

func (man *SLoadbalancerBackendGroupManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.LoadbalancerBackendGroupCreateInput) (*api.LoadbalancerBackendGroupCreateInput, error) {
	lbObj, err := validators.ValidateModel(ctx, userCred, LoadbalancerManager, &input.LoadbalancerId)
	if err != nil {
		return nil, err
	}
	lb := lbObj.(*SLoadbalancer)
	input.StatusStandaloneResourceCreateInput, err = man.SStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusStandaloneResourceCreateInput)
	if err != nil {
		return nil, err
	}

	region, err := lb.GetRegion()
	if err != nil {
		return nil, err
	}
	lbIsManaged := lb.IsManaged()
	for i := 0; i < len(input.Backends); i++ {
		if len(input.Backends[i].BackendType) == 0 {
			input.Backends[i].BackendType = api.LB_BACKEND_GUEST
		}
		if input.Backends[i].Weight < 0 || input.Backends[i].Weight > 256 {
			return nil, httperrors.NewInputParameterError("weight %d not support, only support range 0 ~ 256", input.Backends[i].Weight)
		}
		if input.Backends[i].Port < 1 || input.Backends[i].Port > 65535 {
			return nil, httperrors.NewInputParameterError("port %d not support, only support range 1 ~ 65535", input.Backends[i].Port)
		}
		if len(input.Backends[i].Id) == 0 {
			return nil, httperrors.NewMissingParameterError("Missing backend id")
		}

		var backendRegion *SCloudregion

		switch input.Backends[i].BackendType {
		case api.LB_BACKEND_GUEST:
			guestObj, err := validators.ValidateModel(ctx, userCred, GuestManager, &input.Backends[i].Id)
			if err != nil {
				return nil, err
			}
			guest := guestObj.(*SGuest)
			host, err := guest.GetHost()
			if err != nil {
				return nil, errors.Wrapf(err, "GetHost")
			}
			input.Backends[i].ZoneId = host.ZoneId
			input.Backends[i].HostName = host.Name
			input.Backends[i].Id = guest.Id
			input.Backends[i].Name = guest.Name
			input.Backends[i].ExternalId = guest.ExternalId

			address, err := guest.GetAddress()
			if err != nil {
				return nil, err
			}
			input.Backends[i].Address = address
			backendRegion, _ = host.GetRegion()
		case api.LB_BACKEND_HOST:
			if db.IsAdminAllowCreate(userCred, man).Result.IsDeny() {
				return nil, httperrors.NewForbiddenError("only sysadmin can specify host as backend")
			}
			hostObj, err := validators.ValidateModel(ctx, userCred, HostManager, &input.Backends[i].Id)
			if err != nil {
				return nil, err
			}
			host := hostObj.(*SHost)
			input.Backends[i].Id = host.Id
			input.Backends[i].Name = host.Name
			input.Backends[i].ExternalId = host.ExternalId
			input.Backends[i].Address = host.AccessIp
			backendRegion, _ = host.GetRegion()
		default:
			return nil, httperrors.NewInputParameterError("unexpected backend type %s", input.Backends[i].BackendType)
		}
		if lbIsManaged && backendRegion.Id != region.Id {
			return nil, httperrors.NewInputParameterError("region of backend %d does not match that of lb's", i)
		}
	}
	return region.GetDriver().ValidateCreateLoadbalancerBackendGroupData(ctx, userCred, lb, input)
}

func (lbbg *SLoadbalancerBackendGroup) GetLoadbalancerListenerRules() ([]SLoadbalancerListenerRule, error) {
	q := LoadbalancerListenerRuleManager.Query().Equals("backend_group_id", lbbg.Id)
	rules := []SLoadbalancerListenerRule{}
	err := db.FetchModelObjects(LoadbalancerListenerRuleManager, q, &rules)
	if err != nil {
		return nil, err
	}
	return rules, nil
}

func (lbbg *SLoadbalancerBackendGroup) GetLoadbalancerListeners() ([]SLoadbalancerListener, error) {
	q := LoadbalancerListenerManager.Query().Equals("backend_group_id", lbbg.Id)
	listeners := []SLoadbalancerListener{}
	err := db.FetchModelObjects(LoadbalancerListenerManager, q, &listeners)
	if err != nil {
		return nil, err
	}
	return listeners, nil
}

func (lbbg *SLoadbalancerBackendGroup) GetLoadbalancer() (*SLoadbalancer, error) {
	lb, err := LoadbalancerManager.FetchById(lbbg.LoadbalancerId)
	if err != nil {
		return nil, err
	}
	return lb.(*SLoadbalancer), nil
}

func (llbg *SLoadbalancerBackendGroup) GetRegion() (*SCloudregion, error) {
	loadbalancer, err := llbg.GetLoadbalancer()
	if err != nil {
		return nil, err
	}
	return loadbalancer.GetRegion()
}

func (lbbg *SLoadbalancerBackendGroup) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	loadbalancer, err := lbbg.GetLoadbalancer()
	if err != nil {
		return nil, errors.Wrapf(err, "GetLoadbalancer")
	}
	return loadbalancer.GetIRegion(ctx)
}

func (lbbg *SLoadbalancerBackendGroup) GetBackends() ([]SLoadbalancerBackend, error) {
	backends := make([]SLoadbalancerBackend, 0)
	q := LoadbalancerBackendManager.Query().Equals("backend_group_id", lbbg.GetId())
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

func (lbbg *SLoadbalancerBackendGroup) refCount(man db.IModelManager) (int, error) {
	return man.Query().Equals("backend_group_id", lbbg.Id).CountWithError()
}

func lbbgRefManagers() []db.IModelManager {
	return []db.IModelManager{
		LoadbalancerListenerManager,
	}
}

func (lbbg *SLoadbalancerBackendGroup) getRefManagers() []db.IModelManager {
	// 引用Backend Group的数据库
	return lbbgRefManagers()
}

func (man *SLoadbalancerBackendGroupManager) FilterZeroRefBackendGroup(q *sqlchemy.SQuery) (*sqlchemy.SQuery, error) {
	ids := []string{}
	sq := q.SubQuery()
	rows, err := sq.Query(sq.Field("id")).Rows()
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		var lbbgId string
		err = rows.Scan(&lbbgId)
		if err != nil {
			log.Errorf("Get backendgroup id with scan err: %v", err)
			return nil, err
		}
		ids = append(ids, lbbgId)
	}

	for _, m := range lbbgRefManagers() {
		_ids := m.Query("backend_group_id").In("backend_group_id", ids).SubQuery()
		q = q.NotIn("id", _ids)
	}

	return q, nil
}

func (lbbg *SLoadbalancerBackendGroup) isDefault(ctx context.Context) (bool, error) {
	q := LoadbalancerManager.Query().Equals("backend_group_id", lbbg.GetId()).Equals("id", lbbg.LoadbalancerId)
	count, err := q.CountWithError()
	if err != nil {
		return false, errors.Wrap(err, "loadbalancerBackendGroup.isDefault")
	}
	return count > 0, nil
}

func (lbbg *SLoadbalancerBackendGroup) ValidateDeleteCondition(ctx context.Context, info *api.LoadbalancerBackendGroupDetails) error {
	if gotypes.IsNil(info) {
		info = &api.LoadbalancerBackendGroupDetails{}
		info.IsDefault, _ = lbbg.isDefault(ctx)
		info.LbListenerCount, _ = lbbg.GetListenerCount()
	}
	if info.IsDefault {
		return httperrors.NewResourceBusyError("backend group %s is default backend group", lbbg.Id)
	}
	if info.LbListenerCount > 0 {
		return httperrors.NewResourceBusyError("backend group %s is still referred by %d %s",
			lbbg.Id, info.LbListenerCount, LoadbalancerListenerManager.KeywordPlural())
	}

	return lbbg.SStatusStandaloneResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (man *SLoadbalancerBackendGroupManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LoadbalancerBackendGroupDetails {
	rows := make([]api.LoadbalancerBackendGroupDetails, len(objs))

	stdRows := man.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	lbRows := man.SLoadbalancerResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	lbIds := make([]string, len(objs))
	lbbgIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.LoadbalancerBackendGroupDetails{
			StatusStandaloneResourceDetails: stdRows[i],
			LoadbalancerResourceInfo:        lbRows[i],
		}
		lbbg := objs[i].(*SLoadbalancerBackendGroup)
		lbIds[i] = lbbg.LoadbalancerId
		lbbgIds[i] = lbbg.Id
	}

	lbs := map[string]SLoadbalancer{}
	err := db.FetchStandaloneObjectsByIds(LoadbalancerManager, lbIds, &lbs)
	if err != nil {
		return rows
	}

	defaultLbgIds := []string{}
	virObjs := make([]interface{}, len(objs))
	for i := range rows {
		if lb, ok := lbs[lbIds[i]]; ok {
			virObjs[i] = &lb
			rows[i].ProjectId = lb.ProjectId
			if !utils.IsInStringArray(lb.BackendGroupId, defaultLbgIds) {
				defaultLbgIds = append(defaultLbgIds, lb.BackendGroupId)
			}
		}
	}
	for i := range rows {
		rows[i].IsDefault = utils.IsInStringArray(lbbgIds[i], defaultLbgIds)
	}

	for i := range objs {
		q := LoadbalancerListenerManager.Query().Equals("backend_group_id", lbbgIds[i])
		ownerId, queryScope, err, _ := db.FetchCheckQueryOwnerScope(ctx, userCred, query, LoadbalancerListenerManager, policy.PolicyActionList, true)
		if err != nil {
			log.Errorf("FetchCheckQueryOwnerScope error: %v", err)
			return rows
		}

		q = LoadbalancerListenerManager.FilterByOwner(ctx, q, LoadbalancerListenerManager, userCred, ownerId, queryScope)
		rows[i].LbListenerCount, _ = q.CountWithError()
	}

	return rows
}

func (lbbg *SLoadbalancerBackendGroup) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbbg.SStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	input := &api.LoadbalancerBackendGroupCreateInput{}
	data.Unmarshal(input)
	for i := range input.Backends {
		backend := &SLoadbalancerBackend{
			BackendId:   input.Backends[i].Id,
			BackendType: input.Backends[i].BackendType,
			BackendRole: input.Backends[i].BackendRole,
			Weight:      input.Backends[i].Weight,
			Address:     input.Backends[i].Address,
			Port:        input.Backends[i].Port,
		}
		backend.BackendGroupId = lbbg.Id
		backend.Status = api.LB_STATUS_ENABLED
		backend.Name = fmt.Sprintf("%s-%s-%s", lbbg.Name, backend.BackendType, backend.Name)
		backend.SetModelManager(LoadbalancerBackendManager, backend)
		LoadbalancerBackendManager.TableSpec().Insert(ctx, backend)
	}
	lbbg.StartLoadBalancerBackendGroupCreateTask(ctx, userCred, "")
}

func (lbbg *SLoadbalancerBackendGroup) StartLoadBalancerBackendGroupCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) {
	lbbg.SetStatus(ctx, userCred, api.LB_CREATING, "")
	err := func() error {
		task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerLoadbalancerBackendGroupCreateTask", lbbg, userCred, nil, parentTaskId, "", nil)
		if err != nil {
			return errors.Wrapf(err, "NewTask")
		}
		return task.ScheduleRun(nil)
	}()
	if err != nil {
		lbbg.SetStatus(ctx, userCred, api.LB_CREATE_FAILED, err.Error())
	}
}

func (self *SLoadbalancerBackendGroup) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	backends, err := self.GetBackends()
	if err != nil {
		return err
	}
	for i := range backends {
		err := backends[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "RealDelete backend %s", backends[i].Id)
		}
	}
	return self.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (lbbg *SLoadbalancerBackendGroup) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	parasm := jsonutils.NewDict()
	parasm.Add(jsonutils.JSONTrue, "purge")
	return nil, lbbg.StartLoadBalancerBackendGroupDeleteTask(ctx, userCred, parasm, "")
}

func (lbbg *SLoadbalancerBackendGroup) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	lbbg.SetStatus(ctx, userCred, api.LB_STATUS_DELETING, "")
	return lbbg.StartLoadBalancerBackendGroupDeleteTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (lbbg *SLoadbalancerBackendGroup) StartLoadBalancerBackendGroupDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerBackendGroupDeleteTask", lbbg, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (lbbg *SLoadbalancerBackendGroup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (lbbg *SLoadbalancerBackendGroup) GetListener() *SLoadbalancerListener {
	ret := &SLoadbalancerListener{}
	err := LoadbalancerListenerManager.Query().Equals("backend_group_id", lbbg.Id).First(ret)
	if err != nil {
		return nil
	}
	return ret
}

func (lbbg *SLoadbalancerBackendGroup) GetListenerCount() (int, error) {
	return LoadbalancerListenerManager.Query().Equals("backend_group_id", lbbg.Id).CountWithError()
}

func (lbbg *SLoadbalancerBackendGroup) GetBackendsParams() ([]cloudprovider.SLoadbalancerBackend, error) {
	backends, err := lbbg.GetBackends()
	if err != nil {
		return nil, err
	}

	ret := make([]cloudprovider.SLoadbalancerBackend, len(backends))
	for i := range backends {
		b := backends[i]

		externalId := ""
		guest := b.GetGuest()
		if guest != nil {
			externalId = guest.GetExternalId()
		}

		ret[i] = cloudprovider.SLoadbalancerBackend{
			Weight:      b.Weight,
			Port:        b.Port,
			ID:          b.Id,
			Name:        b.Name,
			ExternalID:  externalId,
			BackendType: b.BackendType,
			BackendRole: b.BackendRole,
			Address:     b.Address,
		}
	}

	return ret, nil
}

func (lbbg *SLoadbalancerBackendGroup) GetICloudLoadbalancerBackendGroup(ctx context.Context) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	if len(lbbg.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}

	lb, err := lbbg.GetLoadbalancer()
	if err != nil {
		return nil, errors.Wrapf(err, "GetLoadbalacer")
	}

	iregion, err := lb.GetIRegion(ctx)
	if err != nil {
		return nil, err
	}

	ilb, err := iregion.GetILoadBalancerById(lb.GetExternalId())
	if err != nil {
		return nil, err
	}

	ilbbg, err := ilb.GetILoadBalancerBackendGroupById(lbbg.ExternalId)
	if err != nil {
		return nil, err
	}

	return ilbbg, nil
}

func (lb *SLoadbalancer) SyncLoadbalancerBackendgroups(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, exts []cloudprovider.ICloudLoadbalancerBackendGroup) ([]SLoadbalancerBackendGroup, []cloudprovider.ICloudLoadbalancerBackendGroup, compare.SyncResult) {
	lockman.LockRawObject(ctx, LoadbalancerBackendGroupManager.Keyword(), lb.Id)
	defer lockman.ReleaseRawObject(ctx, LoadbalancerBackendGroupManager.Keyword(), lb.Id)

	localLbgs := []SLoadbalancerBackendGroup{}
	remoteLbbgs := []cloudprovider.ICloudLoadbalancerBackendGroup{}
	syncResult := compare.SyncResult{}

	dbRes, err := lb.GetLoadbalancerBackendgroups()
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	removed := []SLoadbalancerBackendGroup{}
	commondb := []SLoadbalancerBackendGroup{}
	commonext := []cloudprovider.ICloudLoadbalancerBackendGroup{}
	added := []cloudprovider.ICloudLoadbalancerBackendGroup{}

	err = compare.CompareSets(dbRes, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].syncRemove(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
			continue
		}
		syncResult.Delete()
	}
	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudLoadbalancerBackendgroup(ctx, userCred, lb, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
			continue
		}
		localLbgs = append(localLbgs, commondb[i])
		remoteLbbgs = append(remoteLbbgs, commonext[i])
		syncResult.Update()
	}
	for i := 0; i < len(added); i++ {
		lbbg, err := lb.newFromCloudLoadbalancerBackendgroup(ctx, userCred, added[i])
		if err != nil {
			syncResult.AddError(err)
			continue
		}
		localLbgs = append(localLbgs, *lbbg)
		remoteLbbgs = append(remoteLbbgs, added[i])
		syncResult.Add()
	}
	return localLbgs, remoteLbbgs, syncResult
}

func (lbbg *SLoadbalancerBackendGroup) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbbg)
	defer lockman.ReleaseObject(ctx, lbbg)

	err := lbbg.ValidateDeleteCondition(ctx, nil)
	if err != nil { // cannot delete
		lbbg.SetStatus(ctx, userCred, api.LB_STATUS_UNKNOWN, "sync to delete")
		return errors.Wrapf(err, "ValidateDeleteCondition")
	}
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    lbbg,
		Action: notifyclient.ActionSyncDelete,
	})
	return lbbg.RealDelete(ctx, userCred)
}

func (lbbg *SLoadbalancerBackendGroup) SyncWithCloudLoadbalancerBackendgroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	lb *SLoadbalancer,
	ext cloudprovider.ICloudLoadbalancerBackendGroup,
) error {
	diff, err := db.UpdateWithLock(ctx, lbbg, func() error {
		lbbg.Type = ext.GetType()
		lbbg.Status = ext.GetStatus()
		return nil
	})
	if err != nil {
		return err
	}
	if len(diff) > 0 {
		notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
			Obj:    lbbg,
			Action: notifyclient.ActionSyncUpdate,
		})
	}
	if account := lb.GetCloudaccount(); account != nil {
		syncMetadata(ctx, userCred, lbbg, ext, account.ReadOnly)
	}
	db.OpsLog.LogSyncUpdate(lbbg, diff, userCred)

	if ext.IsDefault() {
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

func (lb *SLoadbalancer) newFromCloudLoadbalancerBackendgroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ext cloudprovider.ICloudLoadbalancerBackendGroup,
) (*SLoadbalancerBackendGroup, error) {
	lbbg := &SLoadbalancerBackendGroup{}
	lbbg.SetModelManager(LoadbalancerBackendGroupManager, lbbg)

	lbbg.LoadbalancerId = lb.Id
	lbbg.ExternalId = ext.GetGlobalId()

	lbbg.Type = ext.GetType()
	lbbg.Status = ext.GetStatus()
	lbbg.Name = ext.GetName()

	err := LoadbalancerBackendGroupManager.TableSpec().Insert(ctx, lbbg)
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    lbbg,
		Action: notifyclient.ActionSyncCreate,
	})

	db.OpsLog.LogEvent(lbbg, db.ACT_CREATE, lbbg.GetShortDesc(ctx), userCred)

	if ext.IsDefault() {
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

func (manager *SLoadbalancerBackendGroupManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}

	if keys.ContainsAny(manager.SLoadbalancerResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SLoadbalancerResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SLoadbalancerResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}

func (man *SLoadbalancerBackendGroupManager) InitializeData() error {
	return nil
}

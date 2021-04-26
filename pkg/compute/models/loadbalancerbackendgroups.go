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
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rand"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SLoadbalancerBackendGroupManager struct {
	SLoadbalancerLogSkipper
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SLoadbalancerResourceBaseManager
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
	LoadbalancerBackendGroupManager.SetVirtualObject(LoadbalancerBackendGroupManager)
}

type SLoadbalancerBackendGroup struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	SLoadbalancerResourceBase `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	Type string `width:"36" charset:"ascii" nullable:"false" list:"user" default:"normal" create:"optional"`

	//LoadbalancerId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
}

func (man *SLoadbalancerBackendGroupManager) pendingDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential, q *sqlchemy.SQuery) {
	lbbgs := []SLoadbalancerBackendGroup{}
	db.FetchModelObjects(man, q, &lbbgs)
	for _, lbbg := range lbbgs {
		lbbg.LBPendingDelete(ctx, userCred)
	}
}

// 负载均衡后端服务器组列表
func (man *SLoadbalancerBackendGroupManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerBackendGroupListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}

	q, err = man.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = man.SLoadbalancerResourceBaseManager.ListItemFilter(ctx, q, userCred, query.LoadbalancerFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLoadbalancerResourceBaseManager.ListItemFilter")
	}

	// userProjId := userCred.GetProjectId()
	/*data := jsonutils.Marshal(query).(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "loadbalancer", ModelKeyword: "loadbalancer", OwnerId: userCred},
		{Key: "cloudregion", ModelKeyword: "cloudregion", OwnerId: userCred},
		{Key: "manager", ModelKeyword: "cloudprovider", OwnerId: userCred},
	})
	if err != nil {
		return nil, err
	}*/

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

	q, err = man.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SLoadbalancerResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.LoadbalancerFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLoadbalancerResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (man *SLoadbalancerBackendGroupManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SLoadbalancerResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (man *SLoadbalancerBackendGroupManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	lbV := validators.NewModelIdOrNameValidator("loadbalancer", "loadbalancer", nil)
	err := lbV.Validate(data.(*jsonutils.JSONDict))
	if err == nil {
		return lbV.Model.GetOwnerId(), nil
	}
	return man.SVirtualResourceBaseManager.FetchOwnerId(ctx, data)
}

func (man *SLoadbalancerBackendGroupManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	lbV := validators.NewModelIdOrNameValidator("loadbalancer", "loadbalancer", ownerId)
	err := lbV.Validate(data)
	if err != nil {
		return nil, err
	}

	input := apis.VirtualResourceCreateInput{}
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal VirtualResourceCreateInput fail %s", err)
	}
	input, err = man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))

	var (
		lb          = lbV.Model.(*SLoadbalancer)
		lbRegion    = lb.GetRegion()
		lbIsManaged = lb.IsManaged()
		backends    = []cloudprovider.SLoadbalancerBackend{}
	)
	if data.Contains("backends") {
		if err := data.Unmarshal(&backends, "backends"); err != nil {
			return nil, err
		}
		for i := 0; i < len(backends); i++ {
			if len(backends[i].BackendType) == 0 {
				backends[i].BackendType = api.LB_BACKEND_GUEST
			}
			if backends[i].Weight < 0 || backends[i].Weight > 256 {
				return nil, httperrors.NewInputParameterError("weight %d not support, only support range 0 ~ 256", backends[i].Weight)
			}
			if backends[i].Port < 1 || backends[i].Port > 65535 {
				return nil, httperrors.NewInputParameterError("port %d not support, only support range 1 ~ 65535", backends[i].Port)
			}
			if len(backends[i].ID) == 0 {
				return nil, httperrors.NewMissingParameterError("Missing backend id")
			}

			var backendRegion *SCloudregion

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
				backendRegion = host.GetRegion()
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
				backendRegion = host.GetRegion()
			default:
				return nil, httperrors.NewInputParameterError("unexpected backend type %s", backends[i].BackendType)
			}
			if lbIsManaged && backendRegion.Id != lbRegion.Id {
				return nil, httperrors.NewInputParameterError("region of backend %d does not match that of lb's", i)
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

// 腾讯云要求四层监听IP+port唯一
func (lbbg *SLoadbalancerBackendGroup) AllBackendsUnique() error {
	backendGroupQ := LoadbalancerListenerManager.Query("backend_group_id").Equals("loadbalancer_id", lbbg.LoadbalancerId).In("listener_type", []string{api.LB_LISTENER_TYPE_TCP, api.LB_LISTENER_TYPE_UDP}).NotEquals("backend_group_id", lbbg.GetId()).IsFalse("pending_deleted").SubQuery()

	backendsQ1 := []SLoadbalancerBackend{}
	backendsQ2 := []SLoadbalancerBackend{}
	err := LoadbalancerBackendManager.Query().Equals("backend_group_id", lbbg.GetId()).IsFalse("pending_deleted").All(&backendsQ1)
	if err != nil {
		return errors.Wrap(err, "loadbalancerBackendGroup.AllBackendsUnique.Q1")
	}

	err = LoadbalancerBackendManager.Query().In("backend_group_id", backendGroupQ).IsFalse("pending_deleted").All(&backendsQ2)
	if err != nil {
		return errors.Wrap(err, "loadbalancerBackendGroup.AllBackendsUnique.Q2")
	}

	for _, b1 := range backendsQ1 {
		for _, b2 := range backendsQ2 {
			if b1.BackendId == b2.BackendId && b1.Port == b2.Port {
				return fmt.Errorf("backend %s with port %d is in used in backendgroup %s", b1.BackendId, b1.Port, b2.BackendGroupId)
			}
		}
	}

	return nil
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
	t := man.TableSpec().Instance()
	pdF := t.Field("pending_deleted")
	return t.Query().IsFalse("deleted").
		Equals("backend_group_id", lbbg.Id).
		Filter(sqlchemy.OR(sqlchemy.IsNull(pdF), sqlchemy.IsFalse(pdF))).
		CountWithError()
}

func lbbgRefManagers() []db.IModelManager {
	return []db.IModelManager{
		LoadbalancerListenerManager,
		LoadbalancerListenerRuleManager,
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

func (lbbg *SLoadbalancerBackendGroup) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (lbbg *SLoadbalancerBackendGroup) isDefault(ctx context.Context) (bool, error) {
	q := LoadbalancerManager.Query().Equals("backend_group_id", lbbg.GetId()).Equals("id", lbbg.LoadbalancerId)
	count, err := q.CountWithError()
	if err != nil {
		return false, errors.Wrap(err, "loadbalancerBackendGroup.isDefault")
	}

	if count == 0 {
		return false, nil
	}

	return true, nil
}

func (lbbg *SLoadbalancerBackendGroup) ValidateDeleteCondition(ctx context.Context) error {
	if ok, err := lbbg.isDefault(ctx); err != nil {
		return httperrors.NewInternalServerError("get isDefault fail %s", err.Error())
	} else {
		if ok {
			return httperrors.NewResourceBusyError("backend group %s is default backend group",
				lbbg.Id)
		}
	}

	return lbbg.ValidatePurgeCondition(ctx)
}

func (lbbg *SLoadbalancerBackendGroup) ValidatePurgeCondition(ctx context.Context) error {
	mans := lbbg.getRefManagers()
	for _, m := range mans {
		n, err := lbbg.refCount(m)
		if err != nil {
			return httperrors.NewInternalServerError("get refCount fail %s", err.Error())
		}
		if n > 0 {
			return httperrors.NewResourceBusyError("backend group %s is still referred by %d %s",
				lbbg.Id, n, m.KeywordPlural())
		}
	}

	return nil
}

func (lbbg *SLoadbalancerBackendGroup) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.LoadbalancerBackendGroupDetails, error) {
	return api.LoadbalancerBackendGroupDetails{}, nil
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

	virtRows := man.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	lbRows := man.SLoadbalancerResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.LoadbalancerBackendGroupDetails{
			VirtualResourceDetails:   virtRows[i],
			LoadbalancerResourceInfo: lbRows[i],
		}
	}

	return rows
}

func (lbbg *SLoadbalancerBackendGroup) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbbg.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
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

func (lbbg *SLoadbalancerBackendGroup) StartHuaweiLoadBalancerBackendGroupCreateTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "HuaweiLoadbalancerLoadbalancerBackendGroupCreateTask", lbbg, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbbg *SLoadbalancerBackendGroup) StartAwsLoadBalancerBackendGroupCreateTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "AwsLoadbalancerLoadbalancerBackendGroupCreateTask", lbbg, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbbg *SLoadbalancerBackendGroup) StartOpenstackLoadBalancerBackendGroupCreateTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "OpenstackLoadbalancerLoadbalancerBackendGroupCreateTask", lbbg, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbbg *SLoadbalancerBackendGroup) LBPendingDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	if lb := lbbg.GetLoadbalancer(); lb != nil && lb.BackendGroupId == lbbg.Id {
		if _, err := db.UpdateWithLock(ctx, lb, func() error {
			lb.BackendGroupId = ""
			return nil
		}); err != nil {
			log.Errorf("loadbalancer %s(%s): clear up backend group %s(%s): %v",
				lb.Name, lb.Id, lbbg.Name, lbbg.Id, err)
			return
		}
	}
	lbbg.pendingDeleteSubs(ctx, userCred)
	lbbg.DoPendingDelete(ctx, userCred)
}

func (lbbg *SLoadbalancerBackendGroup) pendingDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential) {
	subMan := LoadbalancerBackendManager
	ownerId := lbbg.GetOwnerId()

	lockman.LockClass(ctx, subMan, db.GetLockClassKey(subMan, ownerId))
	defer lockman.ReleaseClass(ctx, subMan, db.GetLockClassKey(subMan, ownerId))
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

func (lbbg *SLoadbalancerBackendGroup) GetListener() *SLoadbalancerListener {
	ret := &SLoadbalancerListener{}
	err := LoadbalancerListenerManager.Query().Equals("backend_group_id", lbbg.Id).First(ret)
	if err != nil {
		return nil
	}

	return ret
}

func (lbbg *SLoadbalancerBackendGroup) GetBackendGroupParams() (*cloudprovider.SLoadbalancerBackendGroup, error) {
	backends, err := lbbg.GetBackendsParams()
	if err != nil {
		return &cloudprovider.SLoadbalancerBackendGroup{}, err
	}

	listener := lbbg.GetListener()
	listenerId := ""
	if listener != nil {
		listenerId = listener.ExternalId
	}

	ret := &cloudprovider.SLoadbalancerBackendGroup{
		Name:       lbbg.Name,
		GroupType:  lbbg.Type,
		Backends:   backends,
		ListenerID: listenerId,
	}

	loadbalancer := lbbg.GetLoadbalancer()
	if loadbalancer != nil {
		ret.VpcId = loadbalancer.VpcId
		ret.LoadbalancerID = loadbalancer.ExternalId
	}

	return ret, nil
}

func (lbbg *SLoadbalancerBackendGroup) GetHuaweiBackendGroupParams(lblis *SLoadbalancerListener, lbr *SLoadbalancerListenerRule) (*cloudprovider.SLoadbalancerBackendGroup, error) {
	ret, err := lbbg.GetBackendGroupParams()
	if err != nil {
		return ret, err
	}

	var stickySession *cloudprovider.SLoadbalancerStickySession
	if lblis.StickySession == api.LB_BOOL_ON {
		stickySession = &cloudprovider.SLoadbalancerStickySession{
			StickySession:              lblis.StickySession,
			StickySessionCookie:        lblis.StickySessionCookie,
			StickySessionType:          lblis.StickySessionType,
			StickySessionCookieTimeout: lblis.StickySessionCookieTimeout,
		}
	}

	var healthCheck *cloudprovider.SLoadbalancerHealthCheck
	if lblis.HealthCheck == api.LB_BOOL_ON {
		healthCheck = &cloudprovider.SLoadbalancerHealthCheck{
			HealthCheckType:     lblis.HealthCheckType,
			HealthCheckReq:      lblis.HealthCheckReq,
			HealthCheckExp:      lblis.HealthCheckExp,
			HealthCheck:         lblis.HealthCheck,
			HealthCheckTimeout:  lblis.HealthCheckTimeout,
			HealthCheckDomain:   lblis.HealthCheckDomain,
			HealthCheckHttpCode: lblis.HealthCheckHttpCode,
			HealthCheckURI:      lblis.HealthCheckURI,
			HealthCheckInterval: lblis.HealthCheckInterval,
			HealthCheckRise:     lblis.HealthCheckRise,
			HealthCheckFail:     lblis.HealthCheckFall,
		}
	}

	if lbr != nil {
		ret.ListenerID = lbr.GetExternalId()
	} else {
		ret.ListenerID = lblis.GetExternalId()
	}
	ret.ListenType = lblis.ListenerType
	ret.Scheduler = lblis.Scheduler
	ret.StickySession = stickySession
	ret.HealthCheck = healthCheck

	return ret, nil
}

func (lbbg *SLoadbalancerBackendGroup) GetAwsCachedlbbg() ([]SAwsCachedLbbg, error) {
	ret := []SAwsCachedLbbg{}
	q := AwsCachedLbbgManager.Query().Equals("backend_group_id", lbbg.GetId())
	err := db.FetchModelObjects(AwsCachedLbbgManager, q, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "loadbalancerBackendGroup.GetAwsCachedlbbg")
	}

	return ret, nil
}

func (lbbg *SLoadbalancerBackendGroup) GetHuaweiCachedlbbg() ([]SHuaweiCachedLbbg, error) {
	ret := []SHuaweiCachedLbbg{}
	q := HuaweiCachedLbbgManager.Query().Equals("backend_group_id", lbbg.GetId())
	err := db.FetchModelObjects(HuaweiCachedLbbgManager, q, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "loadbalancerBackendGroup.GetHuaweiCachedlbbg")
	}

	return ret, nil
}

func (lbbg *SLoadbalancerBackendGroup) GetQcloudCachedlbbg() ([]SQcloudCachedLbbg, error) {
	ret := []SQcloudCachedLbbg{}
	q := QcloudCachedLbbgManager.Query().Equals("backend_group_id", lbbg.GetId())
	err := db.FetchModelObjects(QcloudCachedLbbgManager, q, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "loadbalancerBackendGroup.GetQcloudCachedlbbg")
	}

	return ret, nil
}

func (lbbg *SLoadbalancerBackendGroup) GetAwsBackendGroupParams(lblis *SLoadbalancerListener, lbr *SLoadbalancerListenerRule) (*cloudprovider.SLoadbalancerBackendGroup, error) {
	ret, err := lbbg.GetBackendGroupParams()
	if err != nil {
		return ret, err
	}

	healthCheck := &cloudprovider.SLoadbalancerHealthCheck{
		HealthCheckType:     lblis.HealthCheckType,
		HealthCheckReq:      lblis.HealthCheckReq,
		HealthCheckExp:      lblis.HealthCheckExp,
		HealthCheck:         lblis.HealthCheck,
		HealthCheckTimeout:  lblis.HealthCheckTimeout,
		HealthCheckDomain:   lblis.HealthCheckDomain,
		HealthCheckHttpCode: lblis.HealthCheckHttpCode,
		HealthCheckURI:      lblis.HealthCheckURI,
		HealthCheckInterval: lblis.HealthCheckInterval,
		HealthCheckRise:     lblis.HealthCheckRise,
		HealthCheckFail:     lblis.HealthCheckFall,
	}

	ret.ListenerID = lblis.GetExternalId()

	lb := lblis.GetLoadbalancer()
	if lb != nil {
		vpc := lb.GetVpc()
		if vpc != nil {
			ret.VpcId = vpc.GetExternalId()
		} else {
			return nil, fmt.Errorf("loadbalancer %s related vpc not found", lb.GetId())
		}
	}
	ret.ListenType = lblis.ListenerType
	ret.ListenPort = lblis.ListenerPort
	ret.Scheduler = lblis.Scheduler
	ret.HealthCheck = healthCheck
	ret.Name = fmt.Sprintf("%s-%s", ret.Name, rand.String(4))

	return ret, nil
}

func (lbbg *SLoadbalancerBackendGroup) GetQcloudBackendGroupParams(lblis *SLoadbalancerListener, lbr *SLoadbalancerListenerRule) (*cloudprovider.SLoadbalancerBackendGroup, error) {
	ret, err := lbbg.GetBackendGroupParams()
	if err != nil {
		return ret, err
	}

	if lbr != nil {
		ret.ListenerID = lbr.GetExternalId()
	} else {
		ret.ListenerID = lblis.GetExternalId()
	}

	return ret, nil
}

func (lbbg *SLoadbalancerBackendGroup) GetOpenstackBackendGroupParams(lblis *SLoadbalancerListener, lbr *SLoadbalancerListenerRule) (*cloudprovider.SLoadbalancerBackendGroup, error) {
	ret, err := lbbg.GetBackendGroupParams()
	if err != nil {
		return nil, errors.Wrap(err, "lbbg.GetBackendGroupParams()")
	}

	var stickySession *cloudprovider.SLoadbalancerStickySession
	if lblis.StickySession == api.LB_BOOL_ON {
		stickySession = &cloudprovider.SLoadbalancerStickySession{
			StickySession:              lblis.StickySession,
			StickySessionCookie:        lblis.StickySessionCookie,
			StickySessionType:          lblis.StickySessionType,
			StickySessionCookieTimeout: lblis.StickySessionCookieTimeout,
		}
	}

	var healthCheck *cloudprovider.SLoadbalancerHealthCheck
	if lblis.HealthCheck == api.LB_BOOL_ON {
		healthCheck = &cloudprovider.SLoadbalancerHealthCheck{
			HealthCheckType:     lblis.HealthCheckType,
			HealthCheckReq:      lblis.HealthCheckReq,
			HealthCheckExp:      lblis.HealthCheckExp,
			HealthCheck:         lblis.HealthCheck,
			HealthCheckTimeout:  lblis.HealthCheckTimeout,
			HealthCheckDomain:   lblis.HealthCheckDomain,
			HealthCheckHttpCode: lblis.HealthCheckHttpCode,
			HealthCheckURI:      lblis.HealthCheckURI,
			HealthCheckInterval: lblis.HealthCheckInterval,
			HealthCheckRise:     lblis.HealthCheckRise,
			HealthCheckFail:     lblis.HealthCheckFall,
		}
	}

	if lbr != nil {
		ret.ListenerID = lbr.GetExternalId()
	} else {
		ret.ListenerID = lblis.GetExternalId()
	}
	ret.ListenType = lblis.ListenerType
	ret.Scheduler = lblis.Scheduler
	ret.StickySession = stickySession
	ret.HealthCheck = healthCheck

	return ret, nil
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

func (lbbg *SLoadbalancerBackendGroup) GetICloudLoadbalancerBackendGroup() (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	if len(lbbg.ExternalId) == 0 {
		return nil, fmt.Errorf("backendgroup %s has no external id", lbbg.GetId())
	}

	lb := lbbg.GetLoadbalancer()
	if lb == nil {
		return nil, fmt.Errorf("backendgroup %s releated loadbalancer not found", lbbg.GetId())
	}

	iregion, err := lb.GetIRegion()
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

func (man *SLoadbalancerBackendGroupManager) getLoadbalancerBackendgroupsByRegion(regonId string) ([]SLoadbalancerBackendGroup, error) {
	lbbgs := []SLoadbalancerBackendGroup{}
	q := man.Query().Equals("cloudregion_id", regonId).IsFalse("pending_deleted")
	if err := db.FetchModelObjects(man, q, &lbbgs); err != nil {
		log.Errorf("failed to get lbbgs for region: %s error: %v", regonId, err)
		return nil, err
	}
	return lbbgs, nil
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
	syncOwnerId := provider.GetOwnerId()

	lockman.LockRawObject(ctx, "backendgroups", lb.Id)
	defer lockman.ReleaseRawObject(ctx, "backendgroups", lb.Id)

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
		err = commondb[i].SyncWithCloudLoadbalancerBackendgroup(ctx, userCred, lb, commonext[i], provider.GetOwnerId(), provider)
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
		new, err := man.newFromCloudLoadbalancerBackendgroup(ctx, userCred, lb, added[i], syncOwnerId, provider)
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

func (lbbg *SLoadbalancerBackendGroup) SyncWithCloudLoadbalancerBackendgroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	lb *SLoadbalancer,
	extLoadbalancerBackendgroup cloudprovider.ICloudLoadbalancerBackendGroup,
	syncOwnerId mcclient.IIdentityProvider,
	provider *SCloudprovider,
) error {
	diff, err := db.UpdateWithLock(ctx, lbbg, func() error {
		lbbg.Type = extLoadbalancerBackendgroup.GetType()
		lbbg.Status = extLoadbalancerBackendgroup.GetStatus()
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(lbbg, diff, userCred)

	SyncCloudProject(userCred, lbbg, syncOwnerId, extLoadbalancerBackendgroup, provider.Id)

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

func (man *SLoadbalancerBackendGroupManager) newFromCloudLoadbalancerBackendgroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	lb *SLoadbalancer,
	extLoadbalancerBackendgroup cloudprovider.ICloudLoadbalancerBackendGroup,
	syncOwnerId mcclient.IIdentityProvider,
	provider *SCloudprovider,
) (*SLoadbalancerBackendGroup, error) {
	lbbg := &SLoadbalancerBackendGroup{}
	lbbg.SetModelManager(man, lbbg)

	lbbg.LoadbalancerId = lb.Id
	lbbg.ExternalId = extLoadbalancerBackendgroup.GetGlobalId()

	// lbbg.CloudregionId = lb.CloudregionId
	// lbbg.ManagerId = lb.ManagerId

	/*lbbg.constructFieldsFromCloudBackendgroup(lb, extLoadbalancerBackendgroup)
	if lbbg.GetProviderName() != CLOUD_PROVIDER_QCLOUD || len(lbbg.Name) == 0 {

	}*/

	lbbg.Type = extLoadbalancerBackendgroup.GetType()
	lbbg.Status = extLoadbalancerBackendgroup.GetStatus()
	var err = func() error {
		lockman.LockRawObject(ctx, man.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, man.Keyword(), "name")

		newName, err := db.GenerateName(ctx, man, syncOwnerId, extLoadbalancerBackendgroup.GetName())
		if err != nil {
			return err
		}
		lbbg.Name = newName

		return man.TableSpec().Insert(ctx, lbbg)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	SyncCloudProject(userCred, lbbg, syncOwnerId, extLoadbalancerBackendgroup, provider.Id)

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
	q := man.Query().IsNullOrEmpty("loadbalancer_id")
	lbbgs := make([]SLoadbalancerBackendGroup, 0)
	err := db.FetchModelObjects(man, q, &lbbgs)
	if err != nil {
		return errors.Wrap(err, "SLoadbalancerBackendGroupManager.InitializeData")
	}

	for i := range lbbgs {
		lbbg := lbbgs[i]
		_, err = db.UpdateWithLock(context.Background(), &lbbg, func() error {
			lbbg.MarkDelete()
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "SLoadbalancerBackendGroupManager.InitializeData.MarkDelete")
		}
	}

	log.Debugf("SLoadbalancerBackendGroupManager.InitializeData removed %d invalid loadbalancer backendgroup.", len(lbbgs))
	return nil
}

/*
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
}*/

func (manager *SLoadbalancerBackendGroupManager) GetResourceCount() ([]db.SScopeResourceCount, error) {
	virts := manager.Query().IsFalse("pending_deleted")
	return db.CalculateResourceCount(virts, "tenant_id")
}

func (manager *SLoadbalancerBackendGroupManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemExportKeys")
	}

	if keys.ContainsAny(manager.SLoadbalancerResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SLoadbalancerResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SLoadbalancerResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}

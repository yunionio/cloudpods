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
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rand"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=loadbalancerbackend
// +onecloud:swagger-gen-model-plural=loadbalancerbackends
type SLoadbalancerBackendManager struct {
	SLoadbalancerLogSkipper
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SLoadbalancerBackendgroupResourceBaseManager
}

var LoadbalancerBackendManager *SLoadbalancerBackendManager

func init() {
	LoadbalancerBackendManager = &SLoadbalancerBackendManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SLoadbalancerBackend{},
			"loadbalancerbackends_tbl",
			"loadbalancerbackend",
			"loadbalancerbackends",
		),
	}
	LoadbalancerBackendManager.SetVirtualObject(LoadbalancerBackendManager)
}

type SLoadbalancerBackend struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase

	SLoadbalancerBackendgroupResourceBase `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	BackendId   string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	BackendType string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	BackendRole string `width:"36" charset:"ascii" nullable:"false" list:"user" default:"default" create:"optional"`
	Weight      int    `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	Address     string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	Port        int    `nullable:"false" list:"user" create:"required" update:"user"`

	SendProxy string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user" default:"off"`
	Ssl       string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user" default:"off"`
}

func (manager *SLoadbalancerBackendManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeProject
}

func (self *SLoadbalancerBackend) GetOwnerId() mcclient.IIdentityProvider {
	lbbg, err := self.GetLoadbalancerBackendGroup()
	if err != nil {
		return nil
	}
	return lbbg.GetOwnerId()
}

func (manager *SLoadbalancerBackendManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	lbbgId, _ := data.GetString("backend_group_id")
	if len(lbbgId) > 0 {
		lbbg, err := db.FetchById(LoadbalancerBackendGroupManager, lbbgId)
		if err != nil {
			return nil, errors.Wrapf(err, "db.FetchById(LoadbalancerBackendGroupManager, %s)", lbbgId)
		}
		return lbbg.(*SLoadbalancerBackendGroup).GetOwnerId(), nil
	}
	return db.FetchProjectInfo(ctx, data)
}

func (man *SLoadbalancerBackendManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, manager db.FilterByOwnerProvider, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if ownerId != nil {
		sq := LoadbalancerBackendGroupManager.Query("id")
		lb := LoadbalancerManager.Query().SubQuery()
		sq = sq.Join(lb, sqlchemy.Equals(sq.Field("loadbalancer_id"), lb.Field("id")))
		switch scope {
		case rbacscope.ScopeProject:
			sq = sq.Filter(sqlchemy.Equals(lb.Field("tenant_id"), ownerId.GetProjectId()))
			return q.In("backend_group_id", sq.SubQuery())
		case rbacscope.ScopeDomain:
			sq = sq.Filter(sqlchemy.Equals(lb.Field("domain_id"), ownerId.GetProjectDomainId()))
			return q.In("backend_group_id", sq.SubQuery())
		}
	}
	return q
}

// 负载均衡后端列表
func (man *SLoadbalancerBackendManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerBackendListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = man.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = man.SLoadbalancerBackendgroupResourceBaseManager.ListItemFilter(ctx, q, userCred, query.LoadbalancerBackendGroupFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLoadbalancerBackendgroupResourceBaseManager.ListItemFilter")
	}

	data := jsonutils.Marshal(query).(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(ctx, q, data, []*validators.ModelFilterOptions{
		{Key: "backend", ModelKeyword: "server", OwnerId: userCred}, // NOTE extend this when new backend_type was added
	})
	if err != nil {
		return nil, err
	}

	if len(query.BackendType) > 0 {
		q = q.In("backend_type", query.BackendType)
	}
	if len(query.BackendRole) > 0 {
		q = q.In("backend_role", query.BackendRole)
	}
	if len(query.Address) > 0 {
		q = q.In("address", query.Address)
	}
	if len(query.SendProxy) > 0 {
		q = q.In("send_proxy", query.SendProxy)
	}
	if len(query.Ssl) > 0 {
		q = q.In("ssl", query.Ssl)
	}

	return q, nil
}

func (man *SLoadbalancerBackendManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerBackendListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SLoadbalancerBackendgroupResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.LoadbalancerBackendGroupFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLoadbalancerBackendgroupResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (man *SLoadbalancerBackendManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SLoadbalancerBackendgroupResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (man *SLoadbalancerBackendManager) ValidateBackendVpc(lb *SLoadbalancer, guest *SGuest, backendgroup *SLoadbalancerBackendGroup) error {
	region, err := lb.GetRegion()
	if err != nil {
		return err
	}
	requireStatus := region.GetDriver().GetBackendStatusForAdd()
	if !utils.IsInStringArray(guest.Status, requireStatus) {
		return httperrors.NewUnsupportOperationError("%s requires the virtual machine state to be %s before it can be added backendgroup, but current state of the virtual machine is %s", region.GetDriver().GetProvider(), requireStatus, guest.Status)
	}
	vpc, err := guest.GetVpc()
	if err != nil {
		return httperrors.NewBadRequestError("%s", err)
	}
	if len(lb.VpcId) > 0 {
		lbVpc, err := lb.GetVpc()
		if err != nil {
			return err
		}
		if lbVpc != nil && !lbVpc.IsEmulated && vpc.Id != lb.VpcId {
			return httperrors.NewBadRequestError("guest %s(%s) vpc %s(%s) not same as loadbalancer vpc %s", guest.Name, guest.Id, vpc.Name, vpc.Id, lb.VpcId)
		}
		return nil
	}
	backends, err := backendgroup.GetBackends()
	if err != nil {
		return err
	}
	for _, backend := range backends {
		_server, err := GuestManager.FetchById(backend.BackendId)
		if err != nil {
			return httperrors.NewBadRequestError("failed getting guest %s", backend.BackendId)
		}
		server := _server.(*SGuest)
		_vpc, err := server.GetVpc()
		if err != nil {
			return httperrors.NewBadRequestError("%s", err)
		}
		if _vpc.Id != vpc.Id {
			return httperrors.NewBadRequestError("guest %s(%s) vpc %s(%s) not same as vpc %s(%s)", guest.Name, guest.Id, vpc.Name, vpc.Id, _vpc.Name, _vpc.Id)
		}
		if _server.GetId() == guest.Id {
			return httperrors.NewBadRequestError("guest %s(%s) is already in the backendgroup %s(%s)", guest.Name, guest.Id, backendgroup.Name, backendgroup.Id)
		}
	}
	return nil
}

func (man *SLoadbalancerBackendManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject,
	input *api.LoadbalancerBackendCreateInput) (*api.LoadbalancerBackendCreateInput, error) {
	lbbgObj, err := validators.ValidateModel(ctx, userCred, LoadbalancerBackendGroupManager, &input.BackendGroupId)
	if err != nil {
		return nil, err
	}
	lbbg := lbbgObj.(*SLoadbalancerBackendGroup)
	if input.Port < 1 || input.Port > 65535 {
		return input, httperrors.NewInputParameterError("invalid port %d", input.Port)
	}
	if input.Weight < 0 || input.Weight > 100 {
		return input, httperrors.NewInputParameterError("invalid weight %d", input.Weight)
	}
	if len(input.SendProxy) == 0 {
		input.SendProxy = api.LB_SENDPROXY_OFF
	}
	if !utils.IsInStringArray(input.SendProxy, api.LB_SENDPROXY_CHOICES) {
		return input, httperrors.NewInputParameterError("invalid send_proxy %s", input.SendProxy)
	}
	if len(input.Ssl) > 0 && !utils.IsInStringArray(input.Ssl, []string{api.LB_BOOL_ON, api.LB_BOOL_OFF}) {
		return input, httperrors.NewInputParameterError("invalid ssl %s", input.Ssl)
	}
	lb, err := lbbg.GetLoadbalancer()
	if err != nil {
		return nil, errors.Wrapf(err, "GetLoadbalancer")
	}
	if len(input.BackendId) == 0 {
		return nil, httperrors.NewMissingParameterError("backend_id")
	}
	region, err := lb.GetRegion()
	if err != nil {
		return nil, err
	}
	baseName := ""
	switch input.BackendType {
	case api.LB_BACKEND_GUEST:
		guestObj, err := validators.ValidateModel(ctx, userCred, GuestManager, &input.BackendId)
		if err != nil {
			return nil, err
		}
		guest := guestObj.(*SGuest)
		input.Address, err = guest.GetAddress()
		if err != nil {
			return nil, err
		}
		baseName = guest.Name
		host, err := guest.GetHost()
		if err != nil {
			return nil, errors.Wrapf(err, "GetHost")
		}
		hRegion, err := host.GetRegion()
		if err != nil {
			return nil, err
		}
		if hRegion.Id != region.Id {
			return nil, httperrors.NewInputParameterError("region of host %q (%s) != region of loadbalancer %q (%s))",
				host.Name, host.ZoneId, lb.Name, lb.ZoneId)
		}
		if len(lb.ManagerId) == 0 {
			if !utils.IsInStringArray(host.HostType, []string{api.HOST_TYPE_HYPERVISOR, api.HOST_TYPE_ESXI, api.HOST_TYPE_BAREMETAL}) {
				return nil, httperrors.NewInputParameterError("host type of host %q (%s) should be either hypervisor, baremetal or esxi",
					host.Name, host.HostType)
			}
		} else if host.ManagerId != lb.ManagerId {
			return nil, httperrors.NewInputParameterError("manager of host %q (%s) != manager of loadbalancer %q (%s))",
				host.Name, host.ManagerId, lb.Name, lb.ManagerId)
		}
	case api.LB_BACKEND_HOST:
		hostObj, err := validators.ValidateModel(ctx, userCred, HostManager, &input.BackendId)
		if err != nil {
			return nil, err
		}
		host := hostObj.(*SHost)
		if len(host.AccessIp) == 0 {
			return nil, fmt.Errorf("host %s has no access ip", host.GetId())
		}
		hRegion, err := host.GetRegion()
		if err != nil {
			return nil, err
		}
		if hRegion.Id != region.Id {
			return nil, httperrors.NewInputParameterError("region of host %q (%s) != region of loadbalancer %q (%s))",
				host.Name, host.ZoneId, lb.Name, lb.ZoneId)
		}
		if host.ManagerId != lb.ManagerId {
			return nil, httperrors.NewInputParameterError("manager of host %q (%s) != manager of loadbalancer %q (%s))",
				host.Name, host.ManagerId, lb.Name, lb.ManagerId)
		}
		input.Address = host.AccessIp
		baseName = host.Name
	case api.LB_BACKEND_IP:
		_, err := netutils.NewIPV4Addr(input.BackendId)
		if err != nil {
			return nil, err
		}
		input.Address = input.BackendId
		baseName = input.Address
	default:
		return input, httperrors.NewInputParameterError("invalid backend_type %s", input.BackendType)
	}
	if len(input.Name) == 0 {
		input.Name = fmt.Sprintf("%s-%s-%s-%s", lbbg.Name, input.BackendType, baseName, rand.String(4))
	}
	input.StatusStandaloneResourceCreateInput, err = man.SStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusStandaloneResourceCreateInput)
	if err != nil {
		return nil, err
	}
	return region.GetDriver().ValidateCreateLoadbalancerBackendData(ctx, userCred, lb, lbbg, input)
}

func (lbb *SLoadbalancerBackend) GetCloudproviderId() string {
	lbbg, _ := lbb.GetLoadbalancerBackendGroup()
	if lbbg != nil {
		return lbbg.GetCloudproviderId()
	}
	return ""
}

func (lbb *SLoadbalancerBackend) GetLoadbalancerBackendGroup() (*SLoadbalancerBackendGroup, error) {
	backendgroup, err := LoadbalancerBackendGroupManager.FetchById(lbb.BackendGroupId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetLoadbalancerBackendGroup(%s)", lbb.BackendGroupId)
	}
	return backendgroup.(*SLoadbalancerBackendGroup), nil
}

func (lbb *SLoadbalancerBackend) GetGuest() *SGuest {
	guest, err := GuestManager.FetchById(lbb.BackendId)
	if err != nil {
		return nil
	}
	return guest.(*SGuest)
}

func (lbb *SLoadbalancerBackend) GetRegion() (*SCloudregion, error) {
	backendgroup, err := lbb.GetLoadbalancerBackendGroup()
	if err != nil {
		return nil, err
	}
	return backendgroup.GetRegion()
}

func (lbb *SLoadbalancerBackend) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	backendgroup, err := lbb.GetLoadbalancerBackendGroup()
	if err != nil {
		return nil, err
	}
	return backendgroup.GetIRegion(ctx)
}

func (lbb *SLoadbalancerBackend) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.LoadbalancerBackendUpdateInput) (*api.LoadbalancerBackendUpdateInput, error) {
	var err error
	input.StatusStandaloneResourceBaseUpdateInput, err = lbb.SStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.StatusStandaloneResourceBaseUpdateInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBase.ValidateUpdateData")
	}
	region, err := lbb.GetRegion()
	if err != nil {
		return nil, err
	}
	lbbg, err := lbb.GetLoadbalancerBackendGroup()
	if err != nil {
		return nil, err
	}
	return region.GetDriver().ValidateUpdateLoadbalancerBackendData(ctx, userCred, lbbg, input)
}

func (lbb *SLoadbalancerBackend) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbb.SStatusStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)
	if data.Contains("port") || data.Contains("weight") {
		lbb.StartLoadBalancerBackendSyncTask(ctx, userCred, "")
	}
}

func (lbb *SLoadbalancerBackend) StartLoadBalancerBackendSyncTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	lbb.SetStatus(ctx, userCred, api.LB_SYNC_CONF, "")
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerBackendSyncTask", lbb, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbb *SLoadbalancerBackend) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbb.SStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	lbb.SetStatus(ctx, userCred, api.LB_CREATING, "")
	err := lbb.StartLoadBalancerBackendCreateTask(ctx, userCred, "")
	if err != nil {
		log.Errorf("Failed to create loadbalancer backend error: %v", err)
	}
}

func (lbb *SLoadbalancerBackend) getVpc(ctx context.Context) (*SVpc, error) {
	if lbb.BackendType != api.LB_BACKEND_GUEST {
		return nil, nil
	}
	guestM, err := GuestManager.FetchById(lbb.BackendId)
	if err != nil {
		return nil, errors.Wrapf(err, "find guest %s", lbb.BackendId)
	}
	guest := guestM.(*SGuest)
	return guest.GetVpc()
}

func (manager *SLoadbalancerBackendManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LoadbalancerBackendDetails {
	rows := make([]api.LoadbalancerBackendDetails, len(objs))
	stdRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	lbbgRows := manager.SLoadbalancerBackendgroupResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	lbIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.LoadbalancerBackendDetails{
			StatusStandaloneResourceDetails:      stdRows[i],
			LoadbalancerBackendGroupResourceInfo: lbbgRows[i],
		}
		lbIds[i] = rows[i].LoadbalancerId
	}

	lbs := map[string]SLoadbalancer{}
	err := db.FetchStandaloneObjectsByIds(LoadbalancerManager, lbIds, &lbs)
	if err != nil {
		return rows
	}

	virObjs := make([]interface{}, len(objs))
	for i := range rows {
		if lb, ok := lbs[lbIds[i]]; ok {
			virObjs[i] = &lb
			rows[i].ProjectId = lb.ProjectId
		}
	}

	return rows
}

func (lbb *SLoadbalancerBackend) StartLoadBalancerBackendCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerBackendCreateTask", lbb, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (lbb *SLoadbalancerBackend) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SLoadbalancerBackend) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (lbb *SLoadbalancerBackend) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	parasm := jsonutils.NewDict()
	parasm.Add(jsonutils.JSONTrue, "purge")
	return nil, lbb.StartLoadBalancerBackendDeleteTask(ctx, userCred, parasm, "")
}

func (lbb *SLoadbalancerBackend) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	lbb.SetStatus(ctx, userCred, api.LB_STATUS_DELETING, "")
	return lbb.StartLoadBalancerBackendDeleteTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (lbb *SLoadbalancerBackend) StartLoadBalancerBackendDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerBackendDeleteTask", lbb, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbb *SLoadbalancerBackend) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	return lbb.SStatusStandaloneResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (lbbg *SLoadbalancerBackendGroup) SyncLoadbalancerBackends(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, exts []cloudprovider.ICloudLoadbalancerBackend) compare.SyncResult {
	lockman.LockRawObject(ctx, LoadbalancerBackendManager.Keyword(), lbbg.Id)
	defer lockman.ReleaseRawObject(ctx, LoadbalancerBackendManager.Keyword(), lbbg.Id)

	result := compare.SyncResult{}
	dbRes, err := lbbg.GetBackends()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := []SLoadbalancerBackend{}
	commondb := []SLoadbalancerBackend{}
	commonext := []cloudprovider.ICloudLoadbalancerBackend{}
	added := []cloudprovider.ICloudLoadbalancerBackend{}

	err = compare.CompareSets(dbRes, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].syncRemove(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}
	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudLoadbalancerBackend(ctx, userCred, commonext[i], provider)
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}
	for i := 0; i < len(added); i++ {
		_, err := lbbg.newFromCloudLoadbalancerBackend(ctx, userCred, added[i], provider)
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}

func (lbb *SLoadbalancerBackend) constructFieldsFromCloudLoadbalancerBackend(ext cloudprovider.ICloudLoadbalancerBackend, managerId string) error {
	lbb.Status = ext.GetStatus()

	lbb.Weight = ext.GetWeight()
	lbb.Port = ext.GetPort()

	lbb.BackendType = ext.GetBackendType()
	lbb.BackendRole = ext.GetBackendRole()

	ipAddr := ext.GetIpAddress()
	if len(lbb.Address) == 0 || ipAddr != lbb.Address {
		lbb.Address = ipAddr
	}

	if lbb.BackendType == api.LB_BACKEND_GUEST {
		instance, err := db.FetchByExternalIdAndManagerId(GuestManager, ext.GetBackendId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			sq := HostManager.Query().SubQuery()
			return q.Join(sq, sqlchemy.Equals(sq.Field("id"), q.Field("host_id"))).Filter(sqlchemy.Equals(sq.Field("manager_id"), managerId))
		})
		if err != nil {
			// 部分弹性伸缩组实例未同步, 忽略找不到实例错误
			if errors.Cause(err) == sql.ErrNoRows {
				return nil
			}
			return errors.Wrapf(err, "FetchByExternalIdAndManagerId %s", ext.GetBackendId())
		}

		guest := instance.(*SGuest)

		lbb.BackendId = guest.Id
		address, err := guest.GetAddress()
		if err != nil {
			return err
		}
		lbb.Address = address
	}

	return nil
}

func (lbb *SLoadbalancerBackend) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbb)
	defer lockman.ReleaseObject(ctx, lbb)

	err := lbb.ValidateDeleteCondition(ctx, nil)
	if err != nil { // cannot delete
		lbb.SetStatus(ctx, userCred, api.LB_STATUS_UNKNOWN, "sync to delete")
		return errors.Wrapf(err, "ValidateDeleteCondition")
	}
	return lbb.RealDelete(ctx, userCred)
}

func (lbb *SLoadbalancerBackend) SyncWithCloudLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudLoadbalancerBackend, provider *SCloudprovider) error {
	diff, err := db.UpdateWithLock(ctx, lbb, func() error {
		return lbb.constructFieldsFromCloudLoadbalancerBackend(ext, provider.Id)
	})
	if err != nil {
		return err
	}
	if account, _ := provider.GetCloudaccount(); account != nil {
		syncMetadata(ctx, userCred, lbb, ext, account.ReadOnly)
	}
	db.OpsLog.LogSyncUpdate(lbb, diff, userCred)
	return nil
}

func (lbbg *SLoadbalancerBackendGroup) newFromCloudLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudLoadbalancerBackend, provider *SCloudprovider) (*SLoadbalancerBackend, error) {
	lbb := &SLoadbalancerBackend{}
	lbb.SetModelManager(LoadbalancerBackendManager, lbb)

	lbb.BackendGroupId = lbbg.Id
	lbb.ExternalId = ext.GetGlobalId()

	err := lbb.constructFieldsFromCloudLoadbalancerBackend(ext, provider.Id)
	if err != nil {
		return nil, errors.Wrapf(err, "constructFieldsFromCloudLoadbalancerBackend")
	}

	lbb.Name = ext.GetName()
	err = LoadbalancerBackendManager.TableSpec().Insert(ctx, lbb)
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}
	syncMetadata(ctx, userCred, lbb, ext, false)
	db.OpsLog.LogEvent(lbb, db.ACT_CREATE, lbb.GetShortDesc(ctx), userCred)
	return lbb, nil
}

func (manager *SLoadbalancerBackendManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}

	if keys.ContainsAny(manager.SLoadbalancerBackendgroupResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SLoadbalancerBackendgroupResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SLoadbalancerBackendgroupResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}

func (manager *SLoadbalancerBackendManager) InitializeData() error {
	return nil
}

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

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SLoadbalancerBackendManager struct {
	SLoadbalancerLogSkipper
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SLoadbalancerBackendgroupResourceBaseManager
}

var LoadbalancerBackendManager *SLoadbalancerBackendManager

func init() {
	LoadbalancerBackendManager = &SLoadbalancerBackendManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SLoadbalancerBackend{},
			"loadbalancerbackends_tbl",
			"loadbalancerbackend",
			"loadbalancerbackends",
		),
	}
	LoadbalancerBackendManager.SetVirtualObject(LoadbalancerBackendManager)
}

type SLoadbalancerBackend struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	//SManagedResourceBase
	//SCloudregionResourceBase
	SLoadbalancerBackendgroupResourceBase `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	// BackendGroupId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	BackendId   string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	BackendType string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	BackendRole string `width:"36" charset:"ascii" nullable:"false" list:"user" default:"default" create:"optional"`
	Weight      int    `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	Address     string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	Port        int    `nullable:"false" list:"user" create:"required" update:"user"`

	SendProxy string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user" default:"off"`
	Ssl       string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user" default:"off"`
}

func (man *SLoadbalancerBackendManager) pendingDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential, q *sqlchemy.SQuery) {
	lbbs := []SLoadbalancerBackend{}
	db.FetchModelObjects(man, q, &lbbs)
	for _, lbb := range lbbs {
		lbb.DoPendingDelete(ctx, userCred)
	}
}

// 负载均衡后端列表
func (man *SLoadbalancerBackendManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerBackendListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = man.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = man.SLoadbalancerBackendgroupResourceBaseManager.ListItemFilter(ctx, q, userCred, query.LoadbalancerBackendGroupFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLoadbalancerBackendgroupResourceBaseManager.ListItemFilter")
	}

	// userProjId := userCred.GetProjectId()
	data := jsonutils.Marshal(query).(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		// {Key: "backend_group", ModelKeyword: "loadbalancerbackendgroup", OwnerId: userCred},
		{Key: "backend", ModelKeyword: "server", OwnerId: userCred}, // NOTE extend this when new backend_type was added
		// {Key: "cloudregion", ModelKeyword: "cloudregion", OwnerId: userCred},
		// {Key: "manager", ModelKeyword: "cloudprovider", OwnerId: userCred},
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

	q, err = man.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SLoadbalancerBackendgroupResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.LoadbalancerBackendGroupFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLoadbalancerBackendgroupResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (man *SLoadbalancerBackendManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
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
	region := lb.GetRegion()
	if region == nil {
		return httperrors.NewResourceNotFoundError("failed to find region for loadbalancer %s", lb.Name)
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
		lbVpc := lb.GetVpc()
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

func (man *SLoadbalancerBackendManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	backendGroupV := validators.NewModelIdOrNameValidator("backend_group", "loadbalancerbackendgroup", nil)
	if err := backendGroupV.Validate(data.(*jsonutils.JSONDict)); err == nil {
		return backendGroupV.Model.GetOwnerId(), nil
	}
	return man.SVirtualResourceBaseManager.FetchOwnerId(ctx, data)
}

func (man *SLoadbalancerBackendManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	backendGroupV := validators.NewModelIdOrNameValidator("backend_group", "loadbalancerbackendgroup", ownerId)
	if err := backendGroupV.Validate(data); err != nil {
		return nil, err
	}

	backendTypeV := validators.NewStringChoicesValidator("backend_type", api.LB_BACKEND_TYPES)
	if err := backendTypeV.Validate(data); err != nil {
		return nil, err
	}

	backendType := backendTypeV.Value
	backendGroup := backendGroupV.Model.(*SLoadbalancerBackendGroup)
	lb := backendGroup.GetLoadbalancer()
	var backendModel db.IModel

	input := apis.VirtualResourceCreateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal VirtualResourceCreateInput fail %s", err)
	}
	input, err = man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))

	region := lb.GetRegion()
	if region == nil {
		return nil, httperrors.NewResourceNotFoundError("failed to find region for loadbalancer %s", lb.Name)
	}

	ctx = context.WithValue(ctx, "ownerId", ownerId)
	return region.GetDriver().ValidateCreateLoadbalancerBackendData(ctx, userCred, data, backendType, lb, backendGroup, backendModel)
}

func (lbb *SLoadbalancerBackend) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (lbb *SLoadbalancerBackend) GetCloudproviderId() string {
	lbbg := lbb.GetLoadbalancerBackendGroup()
	if lbbg != nil {
		return lbbg.GetCloudproviderId()
	}
	return ""
}

func (lbb *SLoadbalancerBackend) GetLoadbalancerBackendGroup() *SLoadbalancerBackendGroup {
	backendgroup, err := LoadbalancerBackendGroupManager.FetchById(lbb.BackendGroupId)
	if err != nil {
		log.Errorf("failed to find backendgroup for backend %s", lbb.Name)
		return nil
	}
	return backendgroup.(*SLoadbalancerBackendGroup)
}

func (lbb *SLoadbalancerBackend) GetGuest() *SGuest {
	guest, err := GuestManager.FetchById(lbb.BackendId)
	if err != nil {
		return nil
	}
	return guest.(*SGuest)
}

func (lbb *SLoadbalancerBackend) GetRegion() *SCloudregion {
	if backendgroup := lbb.GetLoadbalancerBackendGroup(); backendgroup != nil {
		return backendgroup.GetRegion()
	}
	return nil
}

func (lbb *SLoadbalancerBackend) GetIRegion() (cloudprovider.ICloudRegion, error) {
	if backendgroup := lbb.GetLoadbalancerBackendGroup(); backendgroup != nil {
		return backendgroup.GetIRegion()
	}
	return nil, fmt.Errorf("failed to find region for backend %s", lbb.Name)
}

func (man *SLoadbalancerBackendManager) GetGuestAddress(guest *SGuest) (string, error) {
	gns, err := guest.GetNetworks("")
	if err != nil || len(gns) == 0 {
		return "", fmt.Errorf("guest %s has no network attached", guest.GetId())
	}
	for _, gn := range gns {
		if !gn.IsExit() {
			return gn.IpAddr, nil
		}
	}
	return "", fmt.Errorf("guest %s has no intranet address attached", guest.GetId())
}

func (lbb *SLoadbalancerBackend) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	var err error
	input := apis.VirtualResourceBaseUpdateInput{}
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	input, err = lbb.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBase.ValidateUpdateData")
	}
	data.Update(jsonutils.Marshal(input))

	region := lbb.GetRegion()
	if region == nil {
		return nil, httperrors.NewResourceNotFoundError("failed to found region for loadbalancer backend %s", lbb.Name)
	}
	lbbg := lbb.GetLoadbalancerBackendGroup()
	if lbbg == nil {
		return nil, httperrors.NewResourceNotFoundError("failed to found backendgroup for backend %s(%s)", lbb.Name, lbb.Id)
	}

	data.Set("backend_id", jsonutils.NewString(lbb.BackendId))
	return region.GetDriver().ValidateUpdateLoadbalancerBackendData(ctx, userCred, data, lbbg)
}

func (lbb *SLoadbalancerBackend) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbb.SVirtualResourceBase.PostUpdate(ctx, userCred, query, data)
	if data.Contains("port") || data.Contains("weight") {
		lbb.StartLoadBalancerBackendSyncTask(ctx, userCred, "")
	}
}

func (lbb *SLoadbalancerBackend) StartLoadBalancerBackendSyncTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	lbb.SetStatus(userCred, api.LB_SYNC_CONF, "")
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerBackendSyncTask", lbb, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbb *SLoadbalancerBackend) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbb.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	lbb.SetStatus(userCred, api.LB_CREATING, "")
	if err := lbb.StartLoadBalancerBackendCreateTask(ctx, userCred, ""); err != nil {
		log.Errorf("Failed to create loadbalancer backend error: %v", err)
	}
}

func (lbb *SLoadbalancerBackend) getVpc(ctx context.Context) (*SVpc, error) {
	if lbb.BackendType != api.LB_BACKEND_GUEST {
		return nil, nil
	}
	guestM, err := GuestManager.FetchById(lbb.BackendId)
	if err != nil {
		if err == sql.ErrNoRows {
			theLbbJanitor.Signal()
			return nil, nil
		}
		return nil, errors.Wrapf(err, "find guest %s", lbb.BackendId)
	}
	guest := guestM.(*SGuest)
	vpc, err := guest.GetVpc()
	if err != nil {
		return nil, errors.Wrapf(err, "find guest %s(%s) vpc", guest.Name, guest.Id)
	}
	return vpc, nil
}

func (lbb *SLoadbalancerBackend) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.LoadbalancerBackendDetails, error) {
	return api.LoadbalancerBackendDetails{}, nil
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
	virtRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	lbbgRows := manager.SLoadbalancerBackendgroupResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.LoadbalancerBackendDetails{
			VirtualResourceDetails:               virtRows[i],
			LoadbalancerBackendGroupResourceInfo: lbbgRows[i],
		}
	}
	return rows
}

func (lbb *SLoadbalancerBackend) StartLoadBalancerBackendCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerBackendCreateTask", lbb, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbb *SLoadbalancerBackend) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (lbb *SLoadbalancerBackend) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, lbb, "purge")
}

func (lbb *SLoadbalancerBackend) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	parasm := jsonutils.NewDict()
	parasm.Add(jsonutils.JSONTrue, "purge")
	return nil, lbb.StartLoadBalancerBackendDeleteTask(ctx, userCred, parasm, "")
}

func (lbb *SLoadbalancerBackend) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	lbb.SetStatus(userCred, api.LB_STATUS_DELETING, "")
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

func (man *SLoadbalancerBackendManager) getLoadbalancerBackendsByLoadbalancerBackendgroup(loadbalancerBackendgroup *SLoadbalancerBackendGroup) ([]SLoadbalancerBackend, error) {
	loadbalancerBackends := []SLoadbalancerBackend{}
	q := man.Query().Equals("backend_group_id", loadbalancerBackendgroup.Id)
	q = q.IsFalse("pending_deleted")
	if err := db.FetchModelObjects(man, q, &loadbalancerBackends); err != nil {
		return nil, err
	}
	return loadbalancerBackends, nil
}

func (lbb *SLoadbalancerBackend) ValidateDeleteCondition(ctx context.Context) error {
	return lbb.SVirtualResourceBase.ValidateDeleteCondition(ctx)
}

func (man *SLoadbalancerBackendManager) SyncLoadbalancerBackends(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, loadbalancerBackendgroup *SLoadbalancerBackendGroup, lbbs []cloudprovider.ICloudLoadbalancerBackend, syncRange *SSyncRange) compare.SyncResult {
	syncOwnerId := provider.GetOwnerId()

	lockman.LockRawObject(ctx, "backends", loadbalancerBackendgroup.Id)
	defer lockman.ReleaseRawObject(ctx, "backends", loadbalancerBackendgroup.Id)

	syncResult := compare.SyncResult{}

	dbLbbs, err := man.getLoadbalancerBackendsByLoadbalancerBackendgroup(loadbalancerBackendgroup)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := []SLoadbalancerBackend{}
	commondb := []SLoadbalancerBackend{}
	commonext := []cloudprovider.ICloudLoadbalancerBackend{}
	added := []cloudprovider.ICloudLoadbalancerBackend{}

	err = compare.CompareSets(dbLbbs, lbbs, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].syncRemoveCloudLoadbalancerBackend(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudLoadbalancerBackend(ctx, userCred, commonext[i], syncOwnerId, provider)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		local, err := man.newFromCloudLoadbalancerBackend(ctx, userCred, loadbalancerBackendgroup, added[i], syncOwnerId, provider)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, local, added[i])
			syncResult.Add()
		}
	}
	return syncResult
}

func (lbb *SLoadbalancerBackend) constructFieldsFromCloudLoadbalancerBackend(extLoadbalancerBackend cloudprovider.ICloudLoadbalancerBackend, managerId string) error {
	// lbb.Name = extLoadbalancerBackend.GetName()
	lbb.Status = extLoadbalancerBackend.GetStatus()

	lbb.Weight = extLoadbalancerBackend.GetWeight()
	lbb.Port = extLoadbalancerBackend.GetPort()

	lbb.BackendType = extLoadbalancerBackend.GetBackendType()
	lbb.BackendRole = extLoadbalancerBackend.GetBackendRole()

	instance, err := db.FetchByExternalIdAndManagerId(GuestManager, extLoadbalancerBackend.GetBackendId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		sq := HostManager.Query().SubQuery()
		return q.Join(sq, sqlchemy.Equals(sq.Field("id"), q.Field("host_id"))).Filter(sqlchemy.Equals(sq.Field("manager_id"), managerId))
	})
	if err != nil {
		return err
	}
	guest := instance.(*SGuest)

	lbb.BackendId = guest.Id
	address, err := LoadbalancerBackendManager.GetGuestAddress(guest)
	if err != nil {
		return err
	}
	lbb.Address = address
	return nil
}

func (lbb *SLoadbalancerBackend) syncRemoveCloudLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbb)
	defer lockman.ReleaseObject(ctx, lbb)

	err := lbb.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		err = lbb.SetStatus(userCred, api.LB_STATUS_UNKNOWN, "sync to delete")
	} else {
		// err = lbb.MarkPendingDelete(userCred)
		err = lbb.DoPendingDelete(ctx, userCred)
	}
	return err
}

func (lbb *SLoadbalancerBackend) SyncWithCloudLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, extLoadbalancerBackend cloudprovider.ICloudLoadbalancerBackend, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider) error {
	diff, err := db.UpdateWithLock(ctx, lbb, func() error {
		return lbb.constructFieldsFromCloudLoadbalancerBackend(extLoadbalancerBackend, provider.Id)
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(lbb, diff, userCred)

	SyncCloudProject(userCred, lbb, syncOwnerId, extLoadbalancerBackend, provider.Id)

	return nil
}

func (lbb *SLoadbalancerBackend) GetAwsCachedlbb() ([]SAwsCachedLb, error) {
	ret := []SAwsCachedLb{}
	q := AwsCachedLbManager.Query().Equals("backend_id", lbb.GetId())
	err := db.FetchModelObjects(AwsCachedLbManager, q, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "loadbalancerBackend.GetAwsCachedlbb")
	}

	return ret, nil
}

func (lbb *SLoadbalancerBackend) GetHuaweiCachedlbb() ([]SHuaweiCachedLb, error) {
	ret := []SHuaweiCachedLb{}
	q := HuaweiCachedLbManager.Query().Equals("backend_id", lbb.GetId())
	err := db.FetchModelObjects(HuaweiCachedLbManager, q, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "loadbalancerBackend.GetHuaweiCachedlbb")
	}

	return ret, nil
}

func (man *SLoadbalancerBackendManager) newFromCloudLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, loadbalancerBackendgroup *SLoadbalancerBackendGroup, extLoadbalancerBackend cloudprovider.ICloudLoadbalancerBackend, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider) (*SLoadbalancerBackend, error) {
	lbb := &SLoadbalancerBackend{}
	lbb.SetModelManager(man, lbb)

	lbb.BackendGroupId = loadbalancerBackendgroup.Id
	lbb.ExternalId = extLoadbalancerBackend.GetGlobalId()

	// lbb.CloudregionId = loadbalancerBackendgroup.CloudregionId
	// lbb.ManagerId = loadbalancerBackendgroup.ManagerId

	if err := lbb.constructFieldsFromCloudLoadbalancerBackend(extLoadbalancerBackend, provider.Id); err != nil {
		return nil, err
	}

	var err = func() error {
		lockman.LockRawObject(ctx, man.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, man.Keyword(), "name")

		newName, err := db.GenerateName(ctx, man, syncOwnerId, extLoadbalancerBackend.GetName())
		if err != nil {
			return err
		}
		lbb.Name = newName

		return man.TableSpec().Insert(ctx, lbb)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	SyncCloudProject(userCred, lbb, syncOwnerId, extLoadbalancerBackend, provider.Id)

	db.OpsLog.LogEvent(lbb, db.ACT_CREATE, lbb.GetShortDesc(ctx), userCred)

	return lbb, nil
}

func (manager *SLoadbalancerBackendManager) InitializeData() error {
	/*backends := []SLoadbalancerBackend{}
	q := manager.Query()
	q = q.Filter(sqlchemy.IsNullOrEmpty(q.Field("cloudregion_id")))
	if err := db.FetchModelObjects(manager, q, &backends); err != nil {
		return err
	}
	for i := 0; i < len(backends); i++ {
		backend := &backends[i]
		if group := backend.GetLoadbalancerBackendGroup(); group != nil && len(group.CloudregionId) > 0 {
			_, err := db.Update(backend, func() error {
				backend.CloudregionId = group.CloudregionId
				backend.ManagerId = group.ManagerId
				return nil
			})
			if err != nil {
				log.Errorf("failed to update loadbalancer backend %s cloudregion_id", group.Name)
			}
		}
	}*/
	manager.initializeJanitor()
	return nil
}

func (manager *SLoadbalancerBackendManager) GetResourceCount() ([]db.SScopeResourceCount, error) {
	virts := manager.Query().IsFalse("pending_deleted")
	return db.CalculateResourceCount(virts, "tenant_id")
}

func (manager *SLoadbalancerBackendManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemExportKeys")
	}

	if keys.ContainsAny(manager.SLoadbalancerBackendgroupResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SLoadbalancerBackendgroupResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SLoadbalancerBackendgroupResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}

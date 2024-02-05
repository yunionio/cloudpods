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

	"gopkg.in/fatih/set.v0"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SInterVpcNetworkManager struct {
	db.SEnabledStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
}

var InterVpcNetworkManager *SInterVpcNetworkManager

func init() {
	InterVpcNetworkManager = &SInterVpcNetworkManager{
		SEnabledStatusInfrasResourceBaseManager: db.NewEnabledStatusInfrasResourceBaseManager(
			SInterVpcNetwork{},
			"inter_vpc_networks_tbl",
			"inter_vpc_network",
			"inter_vpc_networks",
		),
	}
	InterVpcNetworkManager.SetVirtualObject(InterVpcNetworkManager)
}

type SInterVpcNetwork struct {
	db.SEnabledStatusInfrasResourceBase
	db.SExternalizedResourceBase
	SManagedResourceBase
}

func (manager *SInterVpcNetworkManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SInterVpcNetworkManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.InterVpcNetworkManagerListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SInfrasResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	if db.NeedOrderQuery([]string{query.OrderByVpcCount}) {
		vpcNetVpcQ := InterVpcNetworkVpcManager.Query()
		vpcNetVpcQ = vpcNetVpcQ.AppendField(vpcNetVpcQ.Field("inter_vpc_network_id"), sqlchemy.COUNT("vpc_count", vpcNetVpcQ.Field("vpc_id")))
		vpcNetVpcQ = vpcNetVpcQ.GroupBy(vpcNetVpcQ.Field("inter_vpc_network_id"))
		vpcNetVpcSQ := vpcNetVpcQ.SubQuery()
		q = q.LeftJoin(vpcNetVpcSQ, sqlchemy.Equals(vpcNetVpcSQ.Field("inter_vpc_network_id"), q.Field("id")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(vpcNetVpcSQ.Field("vpc_count"))
		q = db.OrderByFields(q, []string{query.OrderByVpcCount}, []sqlchemy.IQueryField{q.Field("vpc_count")})
	}
	return q, nil
}

// 列表
func (manager *SInterVpcNetworkManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.InterVpcNetworkListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SEnabledStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SInterVpcNetworkManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SEnabledStatusInfrasResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}

func (manager *SInterVpcNetworkManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.InterVpcNetworkCreateInput,
) (api.InterVpcNetworkCreateInput, error) {
	return input, nil
}

func (self *SInterVpcNetwork) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "InterVpcNetworkCreateTask", self, userCred, params, "", "", nil)
	if err != nil {
		return
	}
	self.SetStatus(ctx, userCred, api.INTER_VPC_NETWORK_STATUS_CREATING, "")
	task.ScheduleRun(nil)
}

func (manager *SInterVpcNetworkManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.InterVpcNetworkDetails {
	rows := make([]api.InterVpcNetworkDetails, len(objs))
	stdRows := manager.SEnabledStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	vpcNetworkIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.InterVpcNetworkDetails{
			EnabledStatusInfrasResourceBaseDetails: stdRows[i],
			ManagedResourceInfo:                    manRows[i],
		}
		vpcNetwork := objs[i].(*SInterVpcNetwork)
		vpcNetworkIds[i] = vpcNetwork.Id
	}

	vpcNetworkVpcs := []SInterVpcNetworkVpc{}
	q := InterVpcNetworkVpcManager.Query().In("inter_vpc_network_id", vpcNetworkIds)
	err := db.FetchModelObjects(InterVpcNetworkVpcManager, q, &vpcNetworkVpcs)
	if err != nil {
		return rows
	}
	vpcMap := map[string][]string{}
	for i := range vpcNetworkVpcs {
		if _, ok := vpcMap[vpcNetworkVpcs[i].InterVpcNetworkId]; !ok {
			vpcMap[vpcNetworkVpcs[i].InterVpcNetworkId] = []string{}
		}
		vpcMap[vpcNetworkVpcs[i].InterVpcNetworkId] = append(vpcMap[vpcNetworkVpcs[i].InterVpcNetworkId], vpcNetworkVpcs[i].VpcId)
	}
	for i := range rows {
		rows[i].VpcCount = len(vpcMap[vpcNetworkIds[i]])
	}
	return rows
}

func (manager *SInterVpcNetworkManager) newFromCloudInterVpcNetwork(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudInterVpcNetwork, provider *SCloudprovider) (*SInterVpcNetwork, error) {
	externalVpcIds, err := ext.GetICloudVpcIds()
	if err != nil {
		return nil, errors.Wrapf(err, "GetICloudVpcIds")
	}
	vpcIds := []string{}
	for i := range externalVpcIds {
		vpc, err := db.FetchByExternalIdAndManagerId(VpcManager, externalVpcIds[i], func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			managerQ := CloudproviderManager.Query("id").Equals("provider", provider.Provider)
			return q.In("manager_id", managerQ.SubQuery())
		})
		if err != nil {
			if errors.Cause(err) != sql.ErrNoRows {
				return nil, errors.Wrapf(err, "vpc.FetchByExternalIdAndManagerId(%s)", externalVpcIds[i])
			}
			continue
		}
		vpcIds = append(vpcIds, vpc.GetId())
	}

	interVpcNetwork := &SInterVpcNetwork{}
	interVpcNetwork.SetModelManager(manager, interVpcNetwork)
	interVpcNetwork.Name = ext.GetName()
	interVpcNetwork.Enabled = tristate.True
	interVpcNetwork.Status = ext.GetStatus()
	interVpcNetwork.ManagerId = provider.Id
	interVpcNetwork.ExternalId = ext.GetGlobalId()
	err = manager.TableSpec().Insert(ctx, interVpcNetwork)
	if err != nil {
		return nil, errors.Wrapf(err, "interVpcNetwork.Insert")
	}

	for i := range vpcIds {
		err := interVpcNetwork.AddVpc(ctx, vpcIds[i])
		if err != nil {
			return nil, errors.Wrapf(err, "interVpcNetwork.AddVpc(%s)", vpcIds[i])
		}
	}

	SyncCloudDomain(userCred, interVpcNetwork, provider.GetOwnerId())
	interVpcNetwork.SyncShareState(ctx, userCred, provider.getAccountShareInfo())

	return interVpcNetwork, nil
}

func (self *SInterVpcNetwork) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	task, err := taskman.TaskManager.NewTask(ctx, "InterVpcNetworkDeleteTask", self, userCred, nil, "", "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, api.INTER_VPC_NETWORK_STATUS_DELETING, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SInterVpcNetwork) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := self.RemoveAllVpc(ctx)
	if err != nil {
		return errors.Wrapf(err, "RemoveAllVpc")
	}
	return self.SEnabledStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SInterVpcNetwork) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.InterVpcNetworkSyncstatusInput) (jsonutils.JSONObject, error) {
	return nil, StartResourceSyncStatusTask(ctx, userCred, self, "InterVpcNetworkSyncstatusTask", "")
}

func (self *SInterVpcNetwork) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.RealDelete(ctx, userCred)
}

func (self *SInterVpcNetwork) StartInterVpcNetworkAddVpcTask(ctx context.Context, userCred mcclient.TokenCredential, vpc *SVpc) error {
	data := jsonutils.NewDict()
	data.Set("vpc_id", jsonutils.NewString(vpc.Id))
	task, err := taskman.TaskManager.NewTask(ctx, "InterVpcNetworkAddVpcTask", self, userCred, data, "", "", nil)
	if err != nil {
		return err
	}
	self.SetStatus(ctx, userCred, api.INTER_VPC_NETWORK_STATUS_ADDVPC, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SInterVpcNetwork) PerformAddvpc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.InterVpcNetworkAddVpcInput) (jsonutils.JSONObject, error) {
	if len(input.VpcId) == 0 {
		return nil, httperrors.NewMissingParameterError("vpc_id")
	}
	_vpc, err := validators.ValidateModel(ctx, userCred, VpcManager, &input.VpcId)
	if err != nil {
		return nil, err
	}
	vpc := _vpc.(*SVpc)

	vpcCloudProvider := vpc.GetCloudprovider()
	cloudProvider := self.GetCloudprovider()
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if vpcCloudProvider.Provider != cloudProvider.Provider {
		return nil, httperrors.NewNotSupportedError("vpc joint interVpcNetwork on different cloudprovider is not supported")
	}
	if vpcCloudProvider.AccessUrl != cloudProvider.AccessUrl {
		return nil, httperrors.NewNotSupportedError("vpc joint interVpcNetwork on different cloudEnv is not supported")
	}

	q := InterVpcNetworkVpcManager.Query().Equals("vpc_id", vpc.Id)
	vpcNetworkjoints := []SInterVpcNetworkVpc{}
	err = db.FetchModelObjects(InterVpcNetworkVpcManager, q, &vpcNetworkjoints)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if len(vpcNetworkjoints) > 0 {
		return nil, httperrors.NewInputParameterError("vpc %s already connected to a interVpcNetwork", vpc.Id)
	}

	err = self.StartInterVpcNetworkAddVpcTask(ctx, userCred, vpc)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (self *SInterVpcNetwork) StartInterVpcNetworkRemoveVpcTask(ctx context.Context, userCred mcclient.TokenCredential, vpc *SVpc) error {
	data := jsonutils.NewDict()
	data.Set("vpc_id", jsonutils.NewString(vpc.Id))
	task, err := taskman.TaskManager.NewTask(ctx, "InterVpcNetworkRemoveVpcTask", self, userCred, data, "", "", nil)
	if err != nil {
		return err
	}
	self.SetStatus(ctx, userCred, api.INTER_VPC_NETWORK_STATUS_REMOVEVPC, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SInterVpcNetwork) PerformRemovevpc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.InterVpcNetworkRemoveVpcInput) (jsonutils.JSONObject, error) {
	if len(input.VpcId) == 0 {
		return nil, httperrors.NewMissingParameterError("vpc_id")
	}
	// get vpc
	_vpc, err := VpcManager.FetchByIdOrName(ctx, userCred, input.VpcId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError2("vpc", input.VpcId)
		}
		return nil, httperrors.NewGeneralError(err)
	}

	vpc := _vpc.(*SVpc)

	q := InterVpcNetworkVpcManager.Query().Equals("inter_vpc_network_id", self.Id).Equals("vpc_id", vpc.Id)
	vpcNetworkjoints := []SInterVpcNetworkVpc{}
	err = db.FetchModelObjects(InterVpcNetworkVpcManager, q, &vpcNetworkjoints)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if len(vpcNetworkjoints) == 0 {
		return nil, httperrors.NewInputParameterError("vpc %s is not connected to this interVpcNetwork", vpc.Id)
	}

	err = self.StartInterVpcNetworkRemoveVpcTask(ctx, userCred, vpc)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (self *SInterVpcNetwork) AddVpc(ctx context.Context, vpcId string) error {
	networkVpc := &SInterVpcNetworkVpc{}
	networkVpc.SetModelManager(InterVpcNetworkVpcManager, networkVpc)
	networkVpc.VpcId = vpcId
	networkVpc.InterVpcNetworkId = self.Id
	return InterVpcNetworkVpcManager.TableSpec().Insert(ctx, networkVpc)
}

func (self *SInterVpcNetwork) GetVpcs() ([]SVpc, error) {
	sq := InterVpcNetworkVpcManager.Query("vpc_id").Equals("inter_vpc_network_id", self.Id)
	q := VpcManager.Query().In("id", sq.SubQuery())
	vpcs := []SVpc{}
	err := db.FetchModelObjects(VpcManager, q, &vpcs)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return vpcs, nil
}

func (self *SInterVpcNetwork) RemoveVpc(ctx context.Context, vpcId string) error {
	q := InterVpcNetworkVpcManager.Query().Equals("inter_vpc_network_id", self.Id).Equals("vpc_id", vpcId)
	networkVpcs := []SInterVpcNetworkVpc{}
	err := db.FetchModelObjects(InterVpcNetworkVpcManager, q, &networkVpcs)
	if err != nil {
		return errors.Wrapf(err, "db.FetchModelObjects")
	}
	for i := range networkVpcs {
		err = networkVpcs[i].Delete(ctx, nil)
		if err != nil {
			return errors.Wrap(err, "Delete")
		}
	}
	return nil
}

func (self *SInterVpcNetwork) RemoveAllVpc(ctx context.Context) error {
	q := InterVpcNetworkVpcManager.Query().Equals("inter_vpc_network_id", self.Id)
	networkVpcs := []SInterVpcNetworkVpc{}
	err := db.FetchModelObjects(InterVpcNetworkVpcManager, q, &networkVpcs)
	if err != nil {
		return errors.Wrapf(err, "db.FetchModelObjects")
	}
	for i := range networkVpcs {
		err = networkVpcs[i].Delete(ctx, nil)
		if err != nil {
			return errors.Wrap(err, "Delete")
		}
	}
	return nil
}

func (self *SInterVpcNetwork) SyncWithCloudInterVpcNetwork(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudInterVpcNetwork) error {
	_, err := db.Update(self, func() error {
		self.ExternalId = ext.GetGlobalId()
		self.Status = ext.GetStatus()
		if options.Options.EnableSyncName {
			self.Name = ext.GetName()
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	localVpcs, err := self.GetVpcs()
	if err != nil {
		return errors.Wrapf(err, "GetVpcs")
	}
	externalVpcIds, err := ext.GetICloudVpcIds()
	if err != nil {
		return errors.Wrapf(err, "GetICloudVpcIds")
	}
	remoteVpcIds := []string{}
	manager := self.GetCloudprovider()
	for i := range externalVpcIds {
		vpc, err := db.FetchByExternalIdAndManagerId(VpcManager, externalVpcIds[i], func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			managerQ := CloudproviderManager.Query("id").Equals("provider", manager.Provider)
			return q.In("manager_id", managerQ.SubQuery())
		})
		if err != nil {
			if errors.Cause(err) != sql.ErrNoRows {
				return errors.Wrapf(err, "vpc.FetchByExternalIdAndManagerId(%s)", externalVpcIds[i])
			}
		} else {
			remoteVpcIds = append(remoteVpcIds, vpc.GetId())
		}
	}

	localVpcIdSet := set.New(set.ThreadSafe)
	for i := range localVpcs {
		localVpcIdSet.Add(localVpcs[i].Id)
	}
	remoteVpcIdSet := set.New(set.ThreadSafe)
	for i := range remoteVpcIds {
		remoteVpcIdSet.Add(remoteVpcIds[i])
	}

	for _, del := range set.Difference(localVpcIdSet, remoteVpcIdSet).List() {
		err := self.RemoveVpc(ctx, del.(string))
		if err != nil {
			return errors.Wrapf(err, "self.RemoveVpc %s", del.(string))
		}
	}
	for _, add := range set.Difference(remoteVpcIdSet, localVpcIdSet).List() {
		err := self.AddVpc(ctx, add.(string))
		if err != nil {
			return errors.Wrapf(err, "self.RemoveVpc %s", add.(string))
		}
	}

	return nil
}

/*
func (self *SInterVpcNetwork) GetCloudaccount() (*SCloudaccount, error) {
	account, err := CloudaccountManager.FetchById(self.CloudaccountId)
	if err != nil {
		return nil, errors.Wrapf(err, "CloudaccountManager.FetchById(%s)", self.CloudaccountId)
	}
	return account.(*SCloudaccount), nil
}
*/

func (self *SInterVpcNetwork) GetInterVpcNetworkRouteSets() ([]SInterVpcNetworkRouteSet, error) {
	routes := []SInterVpcNetworkRouteSet{}
	q := InterVpcNetworkRouteSetManager.Query().Equals("inter_vpc_network_id", self.Id)
	err := db.FetchModelObjects(InterVpcNetworkRouteSetManager, q, &routes)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return routes, nil
}

func (self *SInterVpcNetwork) SyncInterVpcNetworkRouteSets(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudInterVpcNetwork, xor bool) compare.SyncResult {
	lockman.LockRawObject(ctx, self.Keyword(), fmt.Sprintf("%s-records", self.Id))
	defer lockman.ReleaseRawObject(ctx, self.Keyword(), fmt.Sprintf("%s-records", self.Id))

	syncResult := compare.SyncResult{}

	iRoutes, err := ext.GetIRoutes()
	if err != nil {
		syncResult.Error(errors.Wrapf(err, "GetIRoutes"))
		return syncResult
	}

	dbRouteSets, err := self.GetInterVpcNetworkRouteSets()
	if err != nil {
		syncResult.Error(errors.Wrapf(err, "GetRouteTableRouteSets"))
		return syncResult
	}

	removed := make([]SInterVpcNetworkRouteSet, 0)
	commondb := make([]SInterVpcNetworkRouteSet, 0)
	commonext := make([]cloudprovider.ICloudInterVpcNetworkRoute, 0)
	added := make([]cloudprovider.ICloudInterVpcNetworkRoute, 0)
	if err := compare.CompareSets(dbRouteSets, iRoutes, &removed, &commondb, &commonext, &added); err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i++ {
		err := removed[i].syncRemoveRouteSet(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	if !xor {
		for i := 0; i < len(commondb); i++ {
			err := commondb[i].syncWithCloudRouteSet(ctx, userCred, self, commonext[i])
			if err != nil {
				syncResult.UpdateError(err)
				continue
			}
			syncResult.Update()
		}
	}

	for i := 0; i < len(added); i++ {
		_, err := InterVpcNetworkRouteSetManager.newRouteSetFromCloud(ctx, userCred, self, added[i])
		if err != nil {
			syncResult.AddError(err)
			continue
		}
		syncResult.Add()
	}

	return syncResult
}

func (self *SInterVpcNetwork) GetProvider(ctx context.Context) (cloudprovider.ICloudProvider, error) {
	provider, err := self.GetCloudprovider().GetProvider(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "self.GetCloudprovider().GetProvider()")
	}
	return provider, nil
}

func (self *SInterVpcNetwork) GetICloudInterVpcNetwork(ctx context.Context) (cloudprovider.ICloudInterVpcNetwork, error) {
	provider, err := self.GetProvider(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "snetwork.GetProvider()")
	}
	iVpcNetwork, err := provider.GetICloudInterVpcNetworkById(self.ExternalId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetICloudInterVpcNetworkById(%s)", self.ExternalId)
	}
	return iVpcNetwork, nil
}

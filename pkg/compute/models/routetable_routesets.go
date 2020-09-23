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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SRouteTableRouteSetManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SRouteTableResourceBaseManager
}

var RouteTableRouteSetManager *SRouteTableRouteSetManager

func init() {
	RouteTableRouteSetManager = &SRouteTableRouteSetManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SRouteTableRouteSet{},
			"route_table_route_sets_tbl",
			"route_table_route_set",
			"route_table_route_sets",
		),
	}
	RouteTableRouteSetManager.SetVirtualObject(RouteTableRouteSetManager)
}

type SRouteTableRouteSet struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SRouteTableResourceBase
	Type         string `width:"36" charset:"ascii" list:"user" update:"domain" create:"optional"`
	Cidr         string `width:"36" charset:"ascii" nullable:"false" list:"domain" update:"domain" create:"domain_required"`
	NextHopType  string `width:"36" charset:"ascii" nullable:"false" list:"domain" update:"domain" create:"domain_required"`
	NextHopId    string `width:"36" charset:"ascii" nullable:"false" list:"domain" update:"domain" create:"domain_required"`
	ExtNextHopId string `width:"36" charset:"ascii" list:"user" update:"domain" create:"optional"`
}

func (manager *SRouteTableRouteSetManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{RouteTableManager},
	}
}

type sRouteSetUniqueValues struct {
	RouteTableId string
	Cidr         string
}

func (manager *SRouteTableRouteSetManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	values := &sRouteSetUniqueValues{}
	data.Unmarshal(values)
	return jsonutils.Marshal(values)
}

func (manager *SRouteTableRouteSetManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	uniq := &sRouteSetUniqueValues{}
	values.Unmarshal(uniq)
	if len(uniq.RouteTableId) > 0 {
		q = q.Equals("route_table_id", uniq.RouteTableId)
	}
	if len(uniq.Cidr) > 0 {
		q = q.Equals("cidr", uniq.Cidr)
	}

	return q
}

func (manager *SRouteTableRouteSetManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.RouteTableRouteSetCreateInput,
) (api.RouteTableRouteSetCreateInput, error) {
	if len(input.Name) == 0 {
		input.Name = input.Cidr
	}
	var err error
	input.StatusStandaloneResourceCreateInput, err = manager.SStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusStandaloneResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ValidateCreateData")
	}
	if !regutils.MatchCIDR(input.Cidr) {
		return input, httperrors.NewInputParameterError("invalid cidr %s", input.Cidr)
	}
	if len(input.RouteTableId) == 0 {
		return input, httperrors.NewMissingParameterError("route_table_id")
	}
	_routeTable, err := RouteTableManager.FetchByIdOrName(userCred, input.RouteTableId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return input, httperrors.NewResourceNotFoundError2("route_table", input.RouteTableId)
		}
		return input, httperrors.NewGeneralError(err)
	}
	routeTable := _routeTable.(*SRouteTable)
	if !routeTable.IsOwner(userCred) && !userCred.HasSystemAdminPrivilege() {
		return input, httperrors.NewForbiddenError("not enough privilege")
	}

	if input.NextHopType != api.Next_HOP_TYPE_VPCPEERING {
		return input, httperrors.NewNotSupportedError("not supported next hop type %s", input.NextHopType)
	}
	if input.NextHopType == api.Next_HOP_TYPE_VPCPEERING {
		_vpcPeer, err := VpcPeeringConnectionManager.FetchByIdOrName(userCred, input.NextHopId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return input, httperrors.NewResourceNotFoundError2("netx_hop_id", input.NextHopId)
			}
			return input, httperrors.NewGeneralError(err)
		}
		vpcPeer := _vpcPeer.(*SVpcPeeringConnection)
		input.ExtNextHopId = vpcPeer.GetExternalId()
	}

	vpc := routeTable.GetVpc()
	account := vpc.GetCloudaccount()
	factory, err := account.GetProviderFactory()
	if err != nil {
		return input, errors.Wrapf(err, "GetProviderFactory")
	}
	if !factory.IsSupportModifyRouteTable() {
		return input, httperrors.NewUnsupportOperationError("Not support modify routetable for provider %s", account.Provider)
	}
	return input, nil
}

func (self *SRouteTableRouteSet) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	routeTable, err := self.GetRouteTable()
	if err != nil {
		log.Errorf("error:%s self.GetRouteTable()", err)
		return
	}

	routeTable.StartRouteTableUpdateTask(ctx, userCred, self, "create")
}

func (manager *SRouteTableRouteSetManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.RouteTableRouteSetListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SRouteTableResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RouteTableFilterList)
	if err != nil {
		return nil, errors.Wrap(err, "SRouteTableResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (self *SRouteTableRouteSet) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.RouteTableRouteSetUpdateInput,
) (api.RouteTableRouteSetUpdateInput, error) {
	var err error
	input.StatusStandaloneResourceBaseUpdateInput, err = self.SStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.StatusStandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStatusStandaloneResourceBase.ValidateUpdateData")
	}
	if !regutils.MatchCIDR(input.Cidr) {
		return input, httperrors.NewInputParameterError("invalid cidr %s", input.Cidr)
	}

	if input.NextHopType != api.Next_HOP_TYPE_VPCPEERING {
		return input, httperrors.NewNotSupportedError("not supported next hop type %s", input.NextHopType)
	}
	if input.NextHopType == api.Next_HOP_TYPE_VPCPEERING {
		_vpcPeer, err := VpcPeeringConnectionManager.FetchByIdOrName(userCred, input.NextHopId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return input, httperrors.NewResourceNotFoundError2("netx_hop_id", input.NextHopId)
			}
			return input, httperrors.NewGeneralError(err)
		}
		vpcPeer := _vpcPeer.(*SVpcPeeringConnection)
		input.ExtNextHopId = vpcPeer.GetExternalId()
	}

	routeTable, err := self.GetRouteTable()
	if err != nil {
		return input, httperrors.NewGeneralError(err)
	}
	if !routeTable.IsOwner(userCred) && !userCred.HasSystemAdminPrivilege() {
		return input, httperrors.NewForbiddenError("not enough privilege")
	}

	vpc, err := self.GetVpc()
	if err != nil {
		return input, httperrors.NewGeneralError(err)
	}

	account := vpc.GetCloudaccount()
	factory, err := account.GetProviderFactory()
	if err != nil {
		return input, errors.Wrapf(err, "GetProviderFactory")
	}
	if !factory.IsSupportModifyRouteTable() {
		return input, httperrors.NewUnsupportOperationError("Not support modify routetable for provider %s", account.Provider)
	}

	return input, nil
}

func (self *SRouteTable) StartRouteTableUpdateTask(ctx context.Context, userCred mcclient.TokenCredential, routeSet *SRouteTableRouteSet, routeSetAction string) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(routeSetAction), "action")
	params.Add(jsonutils.NewString(routeSet.GetId()), "route_table_route_set_id")
	task, err := taskman.TaskManager.NewTask(ctx, "RouteTableUpdateTask", self, userCred, params, "", "", nil)
	if err != nil {
		return errors.Wrap(err, "Start RouteTableUpdateTask fail")
	}
	self.SetStatus(userCred, api.ROUTE_TABLE_UPDATING, "update route")
	task.ScheduleRun(nil)
	return nil
}

func (self *SRouteTableRouteSet) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SStatusStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)

	routeTable, err := self.GetRouteTable()
	if err != nil {
		log.Errorf("error:%s self.GetRouteTable()", err)
		return
	}

	routeTable.StartRouteTableUpdateTask(ctx, userCred, self, "update")
}

func (self *SRouteTableRouteSet) ValidateDeleteCondition(ctx context.Context) error {
	vpc, err := self.GetVpc()
	if err != nil {
		return errors.Wrap(err, "self.GetVpc()")
	}
	account := vpc.GetCloudaccount()
	factory, err := account.GetProviderFactory()
	if err != nil {
		return errors.Wrapf(err, "GetProviderFactory")
	}
	if !factory.IsSupportModifyRouteTable() {
		return httperrors.NewUnsupportOperationError("Not support modify routetable for provider %s", account.Provider)
	}
	return nil
}

func (self *SRouteTableRouteSet) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	routeTable, err := self.GetRouteTable()
	if err != nil {
		return errors.Wrap(err, "self.GetRouteTable()")
	}
	if !routeTable.IsOwner(userCred) && !userCred.HasSystemAdminPrivilege() {
		return errors.Wrap(err, "not enough privilege")
	}
	routeTable.StartRouteTableUpdateTask(ctx, userCred, self, "delete")
	return nil
}

func (self *SRouteTableRouteSet) GetRouteTable() (*SRouteTable, error) {
	routeTable, err := RouteTableManager.FetchById(self.RouteTableId)
	if err != nil {
		return nil, errors.Wrapf(err, "RouteTableManager.FetchById(%s)", self.RouteTableId)
	}
	return routeTable.(*SRouteTable), nil
}

func (self *SRouteTableRouteSet) GetVpc() (*SVpc, error) {
	routeTable, err := self.GetRouteTable()
	if err != nil {
		return nil, errors.Wrap(err, "self.GetRouteTable()")
	}
	return routeTable.GetVpc(), nil
}

func (self *SRouteTableRouteSet) syncRemoveRouteSet(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}
	err = self.RealDelete(ctx, userCred)
	return err
}

func (self *SRouteTableRouteSet) syncWithCloudRouteSet(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, cloudRouteSet cloudprovider.ICloudRoute) error {
	newNextHopId := ""
	if cloudRouteSet.GetNextHopType() == api.Next_HOP_TYPE_VPCPEERING {
		vpc, err := self.GetVpc()
		if err != nil {
			return errors.Wrap(err, "self.GetVpc()")
		}
		vpcPeer, err := vpc.GetVpcPeeringConnectionByExtId(cloudRouteSet.GetNextHop())
		if err == nil {
			newNextHopId = vpcPeer.GetId()
		}
		if len(newNextHopId) == 0 {
			vpcPeer, err := vpc.GetAccepterVpcPeeringConnectionByExtId(cloudRouteSet.GetNextHop())
			if err == nil {
				newNextHopId = vpcPeer.GetId()
			}
		}
	}

	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Name = cloudRouteSet.GetName()
		self.Status = cloudRouteSet.GetStatus()
		self.Type = cloudRouteSet.GetType()
		self.Cidr = cloudRouteSet.GetCidr()
		self.NextHopType = cloudRouteSet.GetNextHopType()
		self.ExtNextHopId = cloudRouteSet.GetNextHop()
		self.NextHopId = newNextHopId
		return nil
	})
	if err != nil {
		return err
	}

	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (manager *SRouteTableRouteSetManager) newRouteSetFromCloud(ctx context.Context, userCred mcclient.TokenCredential, routeTable *SRouteTable, provider *SCloudprovider, cloudRouteSet cloudprovider.ICloudRoute) (*SRouteTableRouteSet, error) {
	routeSet := &SRouteTableRouteSet{
		Type:         cloudRouteSet.GetType(),
		Cidr:         cloudRouteSet.GetCidr(),
		NextHopType:  cloudRouteSet.GetNextHopType(),
		ExtNextHopId: cloudRouteSet.GetNextHop(),
	}

	routeSet.Name = cloudRouteSet.GetName()
	routeSet.Status = cloudRouteSet.GetStatus()
	routeSet.RouteTableId = routeTable.GetId()
	routeSet.ExternalId = cloudRouteSet.GetGlobalId()
	routeSet.SetModelManager(manager, routeSet)
	if cloudRouteSet.GetNextHopType() == api.Next_HOP_TYPE_VPCPEERING {
		vpc := routeTable.GetVpc()
		vpcPeer, err := vpc.GetVpcPeeringConnectionByExtId(cloudRouteSet.GetNextHop())
		if err == nil {
			routeSet.NextHopId = vpcPeer.GetId()
		}
		if len(routeSet.NextHopId) == 0 {
			vpcPeer, err := vpc.GetAccepterVpcPeeringConnectionByExtId(cloudRouteSet.GetNextHop())
			if err == nil {
				routeSet.NextHopId = vpcPeer.GetId()
			}
		}
	}

	var err = func() error {
		basename := routeSetBasename(cloudRouteSet.GetName(), cloudRouteSet.GetCidr())

		lockman.LockClass(ctx, manager, "name")
		defer lockman.ReleaseClass(ctx, manager, "name")

		newName, err := db.GenerateName(ctx, manager, userCred, basename)
		if err != nil {
			return err
		}
		routeSet.Name = newName

		return manager.TableSpec().Insert(ctx, routeSet)
	}()
	if err != nil {
		return nil, err
	}

	db.OpsLog.LogEvent(routeSet, db.ACT_CREATE, routeSet.GetShortDesc(ctx), userCred)
	return routeSet, nil
}

func routeSetBasename(name, cidr string) string {
	if len(name) == 0 {
		return cidr
	}
	return name
}

func (self *SRouteTableRouteSet) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (manager *SRouteTableRouteSetManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}

	q, err = manager.SRouteTableResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SRouteTableResourceBaseManager.ListItemExportKeys")
	}

	return q, nil
}

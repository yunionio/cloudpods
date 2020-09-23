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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/sqlchemy"

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

type SNatSEntryManager struct {
	SNatEntryManager
	SNetworkResourceBaseManager
}

var NatSEntryManager *SNatSEntryManager

func init() {
	NatSEntryManager = &SNatSEntryManager{
		SNatEntryManager: NewNatEntryManager(
			SNatSEntry{},
			"natstables_tbl",
			"natsentry",
			"natsentries",
		),
	}
	NatSEntryManager.SetVirtualObject(NatSEntryManager)
}

type SNatSEntry struct {
	SNatEntry
	SNetworkResourceBase

	IP         string `charset:"ascii" list:"user" create:"required"`
	SourceCIDR string `width:"22" charset:"ascii" list:"user" create:"optional"`
}

func (self *SNatSEntry) GetCloudproviderId() string {
	network, err := self.GetNetwork()
	if err == nil {
		return network.GetCloudproviderId()
	}
	return ""
}

func (self *SNatSEntry) GetNetwork() (*SNetwork, error) {
	if len(self.NetworkId) == 0 {
		return nil, nil
	}
	_network, err := NetworkManager.FetchById(self.NetworkId)
	if err != nil {
		return nil, err
	}
	return _network.(*SNetwork), nil
}

// NAT网关的源地址转换规则列表
func (man *SNatSEntryManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NatSEntryListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SNatEntryManager.ListItemFilter(ctx, q, userCred, query.NatEntryListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNatEntryManager.ListItemFilter")
	}
	netQuery := api.NetworkFilterListInput{
		NetworkFilterListBase: query.NetworkFilterListBase,
	}
	q, err = man.SNetworkResourceBaseManager.ListItemFilter(ctx, q, userCred, netQuery)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.ListItemFilter")
	}

	if len(query.IP) > 0 {
		q = q.In("ip", query.IP)
	}
	if len(query.SourceCIDR) > 0 {
		q = q.In("source_cidr", query.SourceCIDR)
	}

	return q, nil
}

func (manager *SNatSEntryManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NatSEntryListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SNatEntryManager.OrderByExtraFields(ctx, q, userCred, query.NatEntryListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNatEntryManager.OrderByExtraFields")
	}
	netQuery := api.NetworkFilterListInput{
		NetworkFilterListBase: query.NetworkFilterListBase,
	}
	q, err = manager.SNetworkResourceBaseManager.OrderByExtraFields(ctx, q, userCred, netQuery)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SNatSEntryManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SNatEntryManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SNetworkResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (self *SNatSEntry) GetUniqValues() jsonutils.JSONObject {
	return jsonutils.Marshal(map[string]string{"natgateway_id": self.NatgatewayId})
}

func (manager *SNatSEntryManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	natId, _ := data.GetString("natgateway_id")
	return jsonutils.Marshal(map[string]string{"natgateway_id": natId})
}

func (manager *SNatSEntryManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	natId, _ := values.GetString("natgateway_id")
	if len(natId) > 0 {
		q = q.Equals("natgateway_id", natId)
	}
	return q
}

func (man *SNatSEntryManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.SNatSCreateInput) (*api.SNatSCreateInput, error) {
	if len(input.NatgatewayId) == 0 {
		return nil, httperrors.NewMissingParameterError("natgateway_id")
	}
	if len(input.Eip) == 0 {
		return nil, httperrors.NewMissingParameterError("eip")
	}
	if len(input.SourceCidr) == 0 && len(input.NetworkId) == 0 {
		return nil, httperrors.NewMissingParameterError("network_id")
	}
	if len(input.SourceCidr) > 0 && len(input.NetworkId) > 0 {
		return nil, httperrors.NewInputParameterError("source_cidr and network_id conflict")
	}

	_nat, err := validators.ValidateModel(userCred, NatGatewayManager, &input.NatgatewayId)
	if err != nil {
		return nil, err
	}
	nat := _nat.(*SNatGateway)

	if len(input.SourceCidr) > 0 {
		cidr, err := netutils.NewIPV4Prefix(input.SourceCidr)
		if err != nil {
			return nil, httperrors.NewInputParameterError("input.SourceCidr")
		}
		vpc, err := nat.GetVpc()
		if err != nil {
			return nil, errors.Wrapf(err, "GetVpc")
		}
		vpcRange, err := netutils.NewIPV4Prefix(vpc.CidrBlock)
		if err != nil {
			return nil, errors.Wrapf(err, "vpc cidr %s", vpc.CidrBlock)
		}

		if !vpcRange.ToIPRange().ContainsRange(cidr.ToIPRange()) {
			return nil, httperrors.NewInputParameterError("cidr %s is not in range vpc %s", input.SourceCidr, vpc.CidrBlock)
		}
	} else {
		_network, err := validators.ValidateModel(userCred, NetworkManager, &input.NetworkId)
		if err != nil {
			return nil, err
		}
		network := _network.(*SNetwork)
		vpc := network.GetVpc()
		if vpc == nil {
			return nil, httperrors.NewGeneralError(errors.Wrapf(err, "network.GetVpc"))
		}
		if vpc.Id != nat.VpcId {
			return nil, httperrors.NewInputParameterError("network %s not in vpc %s", network.Name, vpc.Name)
		}
	}

	_eip, err := validators.ValidateModel(userCred, ElasticipManager, &input.Eip)
	if err != nil {
		return nil, err
	}
	eip := _eip.(*SElasticip)
	input.Ip = eip.IpAddr

	// check that eip is suitable
	if len(eip.AssociateId) > 0 && eip.AssociateId != input.NatgatewayId {
		return nil, httperrors.NewInputParameterError("eip has been binding to another instance")
	}

	return input, nil
}

func (manager *SNatSEntryManager) SyncNatSTable(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, nat *SNatGateway, extTable []cloudprovider.ICloudNatSEntry) compare.SyncResult {
	syncOwnerId := provider.GetOwnerId()

	lockman.LockRawObject(ctx, "stable", nat.Id)
	defer lockman.ReleaseRawObject(ctx, "stable", nat.Id)

	result := compare.SyncResult{}
	dbNatSTables, err := nat.GetSTable()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SNatSEntry, 0)
	commondb := make([]SNatSEntry, 0)
	commonext := make([]cloudprovider.ICloudNatSEntry, 0)
	added := make([]cloudprovider.ICloudNatSEntry, 0)
	if err := compare.CompareSets(dbNatSTables, extTable, &removed, &commondb, &commonext, &added); err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		err := removed[i].syncRemoveCloudNatSTable(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}

	for i := 0; i < len(commondb); i += 1 {
		err := commondb[i].SyncWithCloudNatSTable(ctx, userCred, commonext[i], syncOwnerId, provider.Id)
		if err != nil {
			result.UpdateError(err)
			continue
		}
		syncMetadata(ctx, userCred, &commondb[i], commonext[i])
		result.Update()
	}

	for i := 0; i < len(added); i += 1 {
		routeTableNew, err := manager.newFromCloudNatSTable(ctx, userCred, syncOwnerId, nat, added[i], provider.Id)
		if err != nil {
			result.AddError(err)
			continue
		}
		syncMetadata(ctx, userCred, routeTableNew, added[i])
		result.Add()
	}
	return result
}

func (self *SNatSEntry) syncRemoveCloudNatSTable(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		return self.SetStatus(userCred, api.VPC_STATUS_UNKNOWN, "sync to delete")
	}
	return self.RealDelete(ctx, userCred)
}

func (self *SNatSEntry) SyncWithCloudNatSTable(ctx context.Context, userCred mcclient.TokenCredential, extEntry cloudprovider.ICloudNatSEntry, syncOwnerId mcclient.IIdentityProvider, managerId string) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = extEntry.GetStatus()
		self.IP = extEntry.GetIP()
		self.SourceCIDR = extEntry.GetSourceCIDR()
		if extNetworkId := extEntry.GetNetworkId(); len(extNetworkId) > 0 {
			network, err := db.FetchByExternalIdAndManagerId(NetworkManager, extNetworkId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				wire := WireManager.Query().SubQuery()
				vpc := VpcManager.Query().SubQuery()
				return q.Join(wire, sqlchemy.Equals(wire.Field("id"), q.Field("wire_id"))).
					Join(vpc, sqlchemy.Equals(vpc.Field("id"), wire.Field("vpc_id"))).
					Filter(sqlchemy.Equals(vpc.Field("manager_id"), managerId))
			})
			if err != nil {
				return err
			}
			self.NetworkId = network.GetId()
		}
		return nil
	})
	if err != nil {
		return err
	}

	SyncCloudDomain(userCred, self, syncOwnerId)

	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (manager *SNatSEntryManager) newFromCloudNatSTable(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, nat *SNatGateway, extEntry cloudprovider.ICloudNatSEntry, managerId string) (*SNatSEntry, error) {
	table := SNatSEntry{}
	table.SetModelManager(manager, &table)

	table.Status = extEntry.GetStatus()
	table.ExternalId = extEntry.GetGlobalId()
	table.IsEmulated = extEntry.IsEmulated()
	table.NatgatewayId = nat.Id

	table.IP = extEntry.GetIP()
	table.SourceCIDR = extEntry.GetSourceCIDR()
	if extNetworkId := extEntry.GetNetworkId(); len(extNetworkId) > 0 {
		network, err := db.FetchByExternalIdAndManagerId(NetworkManager, extNetworkId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			wire := WireManager.Query().SubQuery()
			vpc := VpcManager.Query().SubQuery()
			return q.Join(wire, sqlchemy.Equals(wire.Field("id"), q.Field("wire_id"))).
				Join(vpc, sqlchemy.Equals(vpc.Field("id"), wire.Field("vpc_id"))).
				Filter(sqlchemy.Equals(vpc.Field("manager_id"), managerId))
		})
		if err != nil {
			return nil, err
		}
		table.NetworkId = network.GetId()
	}

	var err = func() error {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		var err error
		table.Name, err = db.GenerateName(ctx, manager, ownerId, extEntry.GetName())
		if err != nil {
			return err
		}

		return manager.TableSpec().Insert(ctx, &table)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	SyncCloudDomain(userCred, &table, ownerId)

	db.OpsLog.LogEvent(&table, db.ACT_CREATE, table.GetShortDesc(ctx), userCred)

	return &table, nil
}

func (self *SNatSEntry) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.NatSEntryDetails, error) {
	return api.NatSEntryDetails{}, nil
}

func (manager *SNatSEntryManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.NatSEntryDetails {
	rows := make([]api.NatSEntryDetails, len(objs))

	netIds := make([]string, len(objs))
	entryRows := manager.SNatEntryManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.NatSEntryDetails{
			NatEntryDetails: entryRows[i],
		}
		netIds[i] = objs[i].(*SNatSEntry).NetworkId
	}

	nets := make(map[string]SNetwork)
	err := db.FetchStandaloneObjectsByIds(NetworkManager, netIds, &nets)
	if err != nil {
		return rows
	}

	for i := range rows {
		if net, ok := nets[netIds[i]]; ok {
			rows[i].Network = api.SimpleNetwork{
				Id:            net.Id,
				Name:          net.Name,
				GuestIpStart:  net.GuestIpStart,
				GuestIpEnd:    net.GuestIpEnd,
				GuestIp6Start: net.GuestIp6Start,
				GuestIp6End:   net.GuestIp6End,
			}
		}
	}

	return rows
}

func (self *SNatSEntry) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	var err = func() error {
		task, err := taskman.TaskManager.NewTask(ctx, "SNatSEntryCreateTask", self, userCred, nil, "", "", nil)
		if err != nil {
			return errors.Wrapf(err, "NewTask")
		}
		return task.ScheduleRun(nil)
	}()
	if err != nil {
		self.SetStatus(userCred, api.NAT_STATUS_CREATE_FAILED, err.Error())
		return
	}
	self.SetStatus(userCred, api.NAT_STATUS_ALLOCATE, "")
}

func (self *SNatSEntry) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteSNatTask(ctx, userCred)
}

func (self *SNatSEntry) StartDeleteSNatTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	var err = func() error {
		task, err := taskman.TaskManager.NewTask(ctx, "SNatSEntryDeleteTask", self, userCred, nil, "", "", nil)
		if err != nil {
			return errors.Wrapf(err, "NewTask")
		}
		return task.ScheduleRun(nil)
	}()
	if err != nil {
		self.SetStatus(userCred, api.NAT_STATUS_DELETE_FAILED, err.Error())
		return err
	}
	self.SetStatus(userCred, api.NAT_STATUS_DELETING, "")
	return nil
}

func (self *SNatSEntry) GetEip() (*SElasticip, error) {
	q := ElasticipManager.Query().Equals("ip_addr", self.IP)
	eips := []SElasticip{}
	err := db.FetchModelObjects(ElasticipManager, q, &eips)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	if len(eips) == 1 {
		return &eips[0], nil
	}
	if len(eips) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, self.IP)
	}
	return nil, errors.Wrapf(cloudprovider.ErrDuplicateId, self.IP)
}

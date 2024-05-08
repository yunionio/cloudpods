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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SLoadbalancernetworkManager struct {
	db.SVirtualJointResourceBaseManager
	SLoadbalancerResourceBaseManager
	SNetworkResourceBaseManager
}

var LoadbalancernetworkManager *SLoadbalancernetworkManager

func init() {
	db.InitManager(func() {
		LoadbalancernetworkManager = &SLoadbalancernetworkManager{
			SVirtualJointResourceBaseManager: db.NewVirtualJointResourceBaseManager(
				SLoadbalancerNetwork{},
				"loadbalancernetworks_tbl",
				"loadbalancernetwork",
				"loadbalancernetworks",
				LoadbalancerManager,
				NetworkManager,
			),
		}
		LoadbalancernetworkManager.SetVirtualObject(LoadbalancernetworkManager)
	})
}

// +onecloud:model-api-gen
type SLoadbalancerNetwork struct {
	db.SVirtualJointResourceBase

	LoadbalancerId string `width:"36" charset:"ascii" nullable:"false" list:"user"`
	NetworkId      string `width:"36" charset:"ascii" nullable:"false" list:"user"`
	IpAddr         string `width:"16" charset:"ascii" list:"user"`
	MacAddr        string `width:"32" charset:"ascii" nullable:"true" list:"user"`
}

func (manager *SLoadbalancernetworkManager) GetMasterFieldName() string {
	return "loadbalancer_id"
}

func (manager *SLoadbalancernetworkManager) GetSlaveFieldName() string {
	return "network_id"
}

func (ln *SLoadbalancerNetwork) Network() *SNetwork {
	network, _ := ln.GetModelManager().FetchById(ln.NetworkId)
	if network != nil {
		return network.(*SNetwork)
	}
	return nil
}

type SLoadbalancerNetworkRequestData struct {
	Loadbalancer *SLoadbalancer
	NetworkId    string
	reserved     bool                      // allocate from reserved
	Address      string                    // the address user intends to use
	strategy     api.IPAllocationDirection // allocate bottom up, top down, randomly
}

type SLoadbalancerNetworkDeleteData struct {
	loadbalancer *SLoadbalancer
	reserve      bool // reserve after delete
}

func (m *SLoadbalancernetworkManager) NewLoadbalancerNetwork(ctx context.Context, userCred mcclient.TokenCredential, req *SLoadbalancerNetworkRequestData) (*SLoadbalancerNetwork, error) {
	networkMan := db.GetModelManager("network").(*SNetworkManager)
	if networkMan == nil {
		return nil, errors.Error("failed getting network manager")
	}
	im, err := networkMan.FetchById(req.NetworkId)
	if err != nil {
		return nil, errors.Wrapf(err, "fetch network %q", req.NetworkId)
	}
	network := im.(*SNetwork)
	ln := &SLoadbalancerNetwork{
		LoadbalancerId: req.Loadbalancer.Id,
		NetworkId:      network.Id,
	}
	ln.SetModelManager(m, ln)

	lockman.LockObject(ctx, network)
	defer lockman.ReleaseObject(ctx, network)
	if req.Loadbalancer.NetworkType == api.LB_NETWORK_TYPE_VPC {
		macAddr, err := GuestnetworkManager.GenerateMac("")
		if err != nil {
			return nil, errors.Wrapf(err, "generate macaddr")
		}
		ln.MacAddr = macAddr
	}

	usedMap := network.GetUsedAddresses(ctx)
	var recentReclaimed map[string]bool
	ipAddr, err := network.GetFreeIP(ctx, userCred,
		usedMap, recentReclaimed, req.Address, req.strategy, req.reserved, api.AddressTypeIPv4)
	if err != nil {
		return nil, errors.Wrap(err, "find a free ip")
	}
	ln.IpAddr = ipAddr
	err = m.TableSpec().Insert(ctx, ln)
	if err != nil {
		// NOTE no need to free ipAddr as GetFreeIP has no side effect
		return nil, err
	}
	return ln, nil
}

func (m *SLoadbalancernetworkManager) DeleteLoadbalancerNetwork(ctx context.Context, userCred mcclient.TokenCredential, req *SLoadbalancerNetworkDeleteData) error {
	q := m.Query().Equals("loadbalancer_id", req.loadbalancer.Id)
	lns := []SLoadbalancerNetwork{}
	err := db.FetchModelObjects(m, q, &lns)
	if err != nil {
		return err
	}
	// TODO pack up errors and continue, then return as a whole
	for _, ln := range lns {
		err := ln.Delete(ctx, userCred)
		if err != nil {
			return err
		}
		if req.reserve && len(ln.IpAddr) > 0 && regutils.MatchIP4Addr(ln.IpAddr) {
			note := fmt.Sprintf("reserved from loadbalancer delete: %s",
				req.loadbalancer.Id)
			reservedIpMan := db.GetModelManager("reservedip").(*SReservedipManager)
			network := ln.Network()
			err := reservedIpMan.ReserveIP(ctx, userCred, network, ln.IpAddr, note, api.AddressTypeIPv4)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *SLoadbalancernetworkManager) syncLoadbalancerNetwork(ctx context.Context, userCred mcclient.TokenCredential, req *SLoadbalancerNetworkRequestData) error {
	_network, err := db.FetchById(NetworkManager, req.NetworkId)
	if err != nil {
		return err
	}
	network := _network.(*SNetwork)
	if len(req.Address) > 0 {
		ip, err := netutils.NewIPV4Addr(req.Address)
		if err != nil {
			return err
		}
		if !network.IsAddressInRange(ip) {
			return fmt.Errorf("address %s is not in the range of network %s(%s)", req.Address, network.Id, network.Name)
		}
	}

	q := m.Query().Equals("loadbalancer_id", req.Loadbalancer.Id).Equals("network_id", req.NetworkId)
	lns := []SLoadbalancerNetwork{}
	if err := db.FetchModelObjects(m, q, &lns); err != nil {
		return err
	}
	if len(lns) == 0 {
		ln := &SLoadbalancerNetwork{LoadbalancerId: req.Loadbalancer.Id, NetworkId: req.NetworkId, IpAddr: req.Address}
		ln.SetModelManager(LoadbalancernetworkManager, ln)
		return m.TableSpec().Insert(ctx, ln)
	}
	for i := 0; i < len(lns); i++ {
		if i == 0 {
			if lns[i].IpAddr != req.Address {
				_, err := db.Update(&lns[i], func() error {
					lns[i].IpAddr = req.Address
					return nil
				})
				if err != nil {
					log.Errorf("update loadbalancer network ipaddr %s error: %v", lns[i].LoadbalancerId, err)
				}
			}
		} else {
			lns[i].Delete(ctx, userCred)
		}
	}
	return nil
}

func (ln *SLoadbalancerNetwork) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, ln)
}

// Detach implements db.IJointModel interface
func (ln *SLoadbalancerNetwork) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, ln)
}

func (manager *SLoadbalancernetworkManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LoadbalancernetworkDetails {
	rows := make([]api.LoadbalancernetworkDetails, len(objs))

	jointRows := manager.SVirtualJointResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	lbIds := make([]string, len(rows))
	netIds := make([]string, len(rows))

	for i := range rows {
		rows[i] = api.LoadbalancernetworkDetails{
			VirtualJointResourceBaseDetails: jointRows[i],
		}
		lbIds[i] = objs[i].(*SLoadbalancerNetwork).LoadbalancerId
		netIds[i] = objs[i].(*SLoadbalancerNetwork).NetworkId
	}

	lbIdMaps, err := db.FetchIdNameMap2(LoadbalancerManager, lbIds)
	if err != nil {
		log.Errorf("db.FetchIdNameMap2 for lbIds fail %s", err)
		return rows
	}
	netIdMaps, err := db.FetchIdNameMap2(NetworkManager, netIds)
	if err != nil {
		log.Errorf("db.FetchIdNameMap2 for netIds fail %s", err)
		return rows
	}

	for i := range rows {
		if name, ok := lbIdMaps[lbIds[i]]; ok {
			rows[i].Loadbalancer = name
		}
		if name, ok := netIdMaps[netIds[i]]; ok {
			rows[i].Network = name
		}
	}

	return rows
}

func totalLBNicCount(
	scope rbacscope.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	rangeObjs []db.IStandaloneModel,
	providers []string,
	brands []string,
	cloudEnv string,
) (int, error) {
	lbs := LoadbalancerManager.Query().SubQuery()
	lbnics := LoadbalancernetworkManager.Query().SubQuery()
	q := lbnics.Query()
	q = q.Join(lbs, sqlchemy.Equals(lbs.Field("id"), lbnics.Field("loadbalancer_id")))

	switch scope {
	case rbacscope.ScopeSystem:
		// do nothing
	case rbacscope.ScopeDomain:
		q = q.Filter(sqlchemy.Equals(lbs.Field("domain_id"), ownerId.GetProjectDomainId()))
	case rbacscope.ScopeProject:
		q = q.Filter(sqlchemy.Equals(lbs.Field("tenant_id"), ownerId.GetProjectId()))
	}
	q = RangeObjectsFilter(q, rangeObjs, nil, lbs.Field("zone_id"), lbs.Field("manager_id"), nil, nil)
	q = CloudProviderFilter(q, lbs.Field("manager_id"), providers, brands, cloudEnv)
	return q.CountWithError()
}

func (manager *SLoadbalancernetworkManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancernetworkListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualJointResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualJointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualJointResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SLoadbalancerResourceBaseManager.ListItemFilter(ctx, q, userCred, query.LoadbalancerFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLoadbalancerResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SNetworkResourceBaseManager.ListItemFilter(ctx, q, userCred, query.NetworkFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.ListItemFilter")
	}

	if len(query.IpAddr) > 0 {
		q = q.In("ip_addr", query.IpAddr)
	}

	return q, nil
}

func (manager *SLoadbalancernetworkManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancernetworkListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualJointResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualJointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualJointResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SLoadbalancerResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.LoadbalancerFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLoadbalancerResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SNetworkResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.NetworkFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SLoadbalancernetworkManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualJointResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SLoadbalancerResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SNetworkResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SLoadbalancernetworkManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualJointResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualJointResourceBaseManager.ListItemExportKeys")
	}

	if keys.ContainsAny(manager.SLoadbalancerResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SLoadbalancerResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SLoadbalancerResourceBaseManager.ListItemExportKeys")
		}
	}

	if keys.ContainsAny(manager.SNetworkResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SNetworkResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SNetworkResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}

func (manager *SLoadbalancernetworkManager) FetchFirstByLbId(
	ctx context.Context,
	lbId string,
) (*SLoadbalancerNetwork, error) {
	ln := &SLoadbalancerNetwork{}
	q := manager.Query().Equals("loadbalancer_id", lbId)
	if err := q.First(ln); err != nil {
		return nil, errors.Wrapf(err, "fetch loadbalancer network for loadbalancer %q", lbId)
	}
	return ln, nil
}

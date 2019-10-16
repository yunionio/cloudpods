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
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SLoadbalancernetworkManager struct {
	db.SVirtualJointResourceBaseManager
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

type SLoadbalancerNetwork struct {
	db.SVirtualJointResourceBase

	LoadbalancerId string `width:"36" charset:"ascii" nullable:"false" list:"admin"`
	NetworkId      string `width:"36" charset:"ascii" nullable:"false" list:"admin"`
	IpAddr         string `width:"16" charset:"ascii" list:"admin"`
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
		return nil, fmt.Errorf("failed getting network manager")
	}
	im, err := networkMan.FetchById(req.NetworkId)
	if err != nil {
		return nil, err
	}
	network := im.(*SNetwork)
	ln := &SLoadbalancerNetwork{
		LoadbalancerId: req.Loadbalancer.Id,
		NetworkId:      network.Id,
	}
	ln.SetModelManager(m, ln)

	lockman.LockObject(ctx, network)
	defer lockman.ReleaseObject(ctx, network)
	usedMap := network.GetUsedAddresses()
	recentReclaimed := map[string]bool{}
	ipAddr, err := network.GetFreeIP(ctx, userCred,
		usedMap, recentReclaimed, req.Address, req.strategy, req.reserved)
	if err != nil {
		return nil, err
	}
	ln.IpAddr = ipAddr
	err = m.TableSpec().Insert(ln)
	if err != nil {
		// NOTE no need to free ipAddr as GetFreeIP has no side effect
		return nil, err
	}
	return ln, err
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
			err := reservedIpMan.ReserveIP(userCred, network, ln.IpAddr, note)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *SLoadbalancernetworkManager) syncLoadbalancerNetwork(ctx context.Context, userCred mcclient.TokenCredential, req *SLoadbalancerNetworkRequestData) error {
	_network, err := db.FetchByExternalId(NetworkManager, req.NetworkId)
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
		return m.TableSpec().Insert(ln)
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

// Master implements db.IJointModel interface
func (ln *SLoadbalancerNetwork) Master() db.IStandaloneModel {
	return db.JointMaster(ln)
}

// Slave implements db.IJointModel interface
func (ln *SLoadbalancerNetwork) Slave() db.IStandaloneModel {
	return db.JointSlave(ln)
}

// Detach implements db.IJointModel interface
func (ln *SLoadbalancerNetwork) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, ln)
}

func (ln *SLoadbalancerNetwork) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := jsonutils.NewDict()
	return db.JointModelExtra(ln, extra)
}

func totalLBNicCount(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider) (int, error) {
	lbs := LoadbalancerManager.Query().SubQuery()
	lbnics := LoadbalancernetworkManager.Query().SubQuery()
	q := lbnics.Query().Join(lbs, sqlchemy.Equals(lbs.Field("id"), lbnics.Field("loadbalancer_id")))
	switch scope {
	case rbacutils.ScopeSystem:
		// do nothing
	case rbacutils.ScopeDomain:
		q = q.Filter(sqlchemy.Equals(lbs.Field("domain_id"), ownerId.GetProjectDomainId()))
	case rbacutils.ScopeProject:
		q = q.Filter(sqlchemy.Equals(lbs.Field("tenant_id"), ownerId.GetProjectId()))
	}
	return q.CountWithError()
}

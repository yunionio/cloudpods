package models

import (
	"context"
	"fmt"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/util/regutils"
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
	})
}

type SLoadbalancerNetwork struct {
	db.SVirtualJointResourceBase

	LoadbalancerId string `width:"36" charset:"ascii" nullable:"false" key_index:"true" list:"admin"`
	NetworkId      string `width:"36" charset:"ascii" nullable:"false" key_index:"true" list:"admin"`
	IpAddr         string `width:"16" charset:"ascii" list:"admin"`
}

func (ln *SLoadbalancerNetwork) Network() *SNetwork {
	network, _ := ln.GetModelManager().FetchById(ln.NetworkId)
	if network != nil {
		return network.(*SNetwork)
	}
	return nil
}

type SLoadbalancerNetworkRequestData struct {
	loadbalancer *SLoadbalancer
	networkId    string
	reserved     bool                   // allocate from reserved
	address      string                 // the address user intends to use
	strategy     IPAddlocationDirection // allocate bottom up, top down, randomly
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
	im, err := networkMan.FetchById(req.networkId)
	if err != nil {
		return nil, err
	}
	network := im.(*SNetwork)
	ln := &SLoadbalancerNetwork{
		LoadbalancerId: req.loadbalancer.Id,
		NetworkId:      network.Id,
	}
	ln.SetModelManager(m)

	lockman.LockObject(ctx, network)
	defer lockman.ReleaseObject(ctx, network)
	usedMap := network.GetUsedAddresses()
	recentReclaimed := map[string]bool{}
	ipAddr, err := network.GetFreeIP(ctx, userCred,
		usedMap, recentReclaimed, req.address, req.strategy, req.reserved)
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
		if req.reserve && regutils.MatchIP4Addr(ln.IpAddr) {
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

func (lbNetwork *SLoadbalancerNetwork) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, lbNetwork)
}

package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SHostnetworkManager struct {
	SHostJointsManager
}

var HostnetworkManager *SHostnetworkManager

func init() {
	db.InitManager(func() {
		HostnetworkManager = &SHostnetworkManager{SHostJointsManager: NewHostJointsManager(SHostnetwork{},
			"baremetalnetworks_tbl", "baremetalnetwork", "baremetalnetworks", NetworkManager)}
	})
}

type SHostnetwork struct {
	SHostJointsBase

	BaremetalId string `width:"36" charset:"ascii" nullable:"false" list:"admin" key_index:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
	NetworkId   string `width:"36" charset:"ascii" nullable:"false" list:"admin" key_index:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
	IpAddr      string `width:"16" charset:"ascii" list:"admin"`                                   // Column(VARCHAR(16, charset='ascii'))
	MacAddr     string `width:"18" charset:"ascii" list:"admin"`                                   // Column(VARCHAR(18, charset='ascii'))
}

func (bn *SHostnetwork) Master() db.IStandaloneModel {
	return db.JointMaster(bn)
}

func (bn *SHostnetwork) Slave() db.IStandaloneModel {
	return db.JointSlave(bn)
}

func (bn *SHostnetwork) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := bn.SHostJointsBase.GetCustomizeColumns(ctx, userCred, query)
	extra = db.JointModelExtra(bn, extra)
	netif := bn.GetNetInterface()
	if netif != nil {
		extra.Add(jsonutils.NewString(netif.NicType), "nic_type")
	}
	return extra
}

func (bn *SHostnetwork) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := bn.SHostJointsBase.GetExtraDetails(ctx, userCred, query)
	return db.JointModelExtra(bn, extra)
}

func (bn *SHostnetwork) GetHost() *SHost {
	master, _ := HostManager.FetchById(bn.BaremetalId)
	if master != nil {
		return master.(*SHost)
	}
	return nil
}

func (bn *SHostnetwork) GetNetwork() *SNetwork {
	slave, _ := NetworkManager.FetchById(bn.NetworkId)
	if slave != nil {
		return slave.(*SNetwork)
	}
	return nil
}

func (bn *SHostnetwork) GetNetInterface() *SNetInterface {
	netIf, _ := NetInterfaceManager.FetchByMac(bn.MacAddr)
	return netIf
}

func (bn *SHostnetwork) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, bn)
}

func (bn *SHostnetwork) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, bn)
}

func (man *SHostnetworkManager) QueryByAddress(addr string) *sqlchemy.SQuery {
	q := HostnetworkManager.Query()
	return q.Filter(sqlchemy.Equals(q.Field("ip_addr"), addr))
}

func (man *SHostnetworkManager) GetHostNetworkByAddress(addr string) *SHostnetwork {
	network := SHostnetwork{}
	err := man.QueryByAddress(addr).First(&network)
	if err != nil {
		return &network
	}
	return nil
}

func (man *SHostnetworkManager) GetNetworkByAddress(addr string) *SNetwork {
	net := man.GetHostNetworkByAddress(addr)
	if net == nil {
		return nil
	}
	return net.GetNetwork()
}

func (man *SHostnetworkManager) GetHostByAddress(addr string) *SHost {
	net := man.GetHostNetworkByAddress(addr)
	if net == nil {
		return nil
	}
	return net.GetHost()
}

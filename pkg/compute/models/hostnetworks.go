package models

import (
	"context"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
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

func (joint *SHostnetwork) Master() db.IStandaloneModel {
	return db.JointMaster(joint)
}

func (joint *SHostnetwork) Slave() db.IStandaloneModel {
	return db.JointSlave(joint)
}

func (self *SHostnetwork) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SHostJointsBase.GetCustomizeColumns(ctx, userCred, query)
	extra = db.JointModelExtra(self, extra)
	netif := self.GetNetInterface()
	if netif != nil {
		extra.Add(jsonutils.NewString(netif.NicType), "nic_type")
	}
	return extra
}

func (self *SHostnetwork) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SHostJointsBase.GetExtraDetails(ctx, userCred, query)
	return db.JointModelExtra(self, extra)
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

func (self *SHostnetwork) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SHostnetwork) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

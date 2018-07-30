package models

import (
	"context"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/pkg/httperrors"
	"github.com/yunionio/sqlchemy"
)

type SHostwireManager struct {
	SHostJointsManager
}

var HostwireManager *SHostwireManager

func init() {
	db.InitManager(func() {
		HostwireManager = &SHostwireManager{SHostJointsManager: NewHostJointsManager(SHostwire{},
			"hostwires_tbl", "hostwire", "hostwires", WireManager)}
	})
}

type SHostwire struct {
	SHostJointsBase

	Bridge    string `width:"16" charset:"ascii" nullable:"false" list:"admin" update:"admin" create:"admin_required"` // Column(VARCHAR(16, charset='ascii'), nullable=False)
	Interface string `width:"16" charset:"ascii" nullable:"false" list:"admin" update:"admin" create:"admin_required"` // Column(VARCHAR(16, charset='ascii'), nullable=False)
	IsMaster  bool   `nullable:"true" default:"false" update:"admin" create:"admin_optional"`                          // Column(Boolean, nullable=True, default=False)
	MacAddr   string `width:"18" charset:"ascii" list:"admin" update:"admin" create:"admin_required"`                  // Column(VARCHAR(18, charset='ascii'))

	HostId string `width:"128" charset:"ascii" nullable:"false" list:"admin" create:"admin_required" key_index:"true"` // = Column(VARCHAR(ID_LENGTH, charset='ascii'), nullable=False)
	WireId string `width:"128" charset:"ascii" nullable:"false" list:"admin" create:"admin_required" key_index:"true"` // Column(VARCHAR(ID_LENGTH, charset='ascii'), nullable=False)
}

func (joint *SHostwire) Master() db.IStandaloneModel {
	return db.JointMaster(joint)
}

func (joint *SHostwire) Slave() db.IStandaloneModel {
	return db.JointSlave(joint)
}

func (self *SHostwire) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SHostJointsBase.GetCustomizeColumns(ctx, userCred, query)
	extra = db.JointModelExtra(self, extra)
	return self.getExtraDetails(extra)
}

func (self *SHostwire) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SHostJointsBase.GetExtraDetails(ctx, userCred, query)
	extra = db.JointModelExtra(self, extra)
	return self.getExtraDetails(extra)
}

func (hw *SHostwire) GetWire() *SWire {
	wire, _ := WireManager.FetchById(hw.WireId)
	if wire != nil {
		return wire.(*SWire)
	}
	return nil
}

func (hw *SHostwire) GetHost() *SHost {
	host, _ := HostManager.FetchById(hw.HostId)
	if host != nil {
		return host.(*SHost)
	}
	return nil
}

func (hw *SHostwire) getExtraDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	wire := hw.GetWire()
	if wire != nil {
		extra.Add(jsonutils.NewInt(int64(wire.Bandwidth)), "bandwidth")
	}
	return extra
}

func (self *SHostwire) GetGuestnicsCount() int {
	guestnics := GuestnetworkManager.Query().SubQuery()
	guests := GuestManager.Query().SubQuery()
	nets := NetworkManager.Query().SubQuery()

	q := guestnics.Query()
	q = q.Join(guests, sqlchemy.AND(sqlchemy.IsFalse(guests.Field("deleted")),
		sqlchemy.Equals(guests.Field("id"), guestnics.Field("guest_id")),
		sqlchemy.Equals(guests.Field("host_id"), self.HostId)))
	q = q.Join(nets, sqlchemy.AND(sqlchemy.IsFalse(nets.Field("deleted")),
		sqlchemy.Equals(nets.Field("id"), guestnics.Field("network_id")),
		sqlchemy.Equals(nets.Field("wire_id"), self.WireId)))

	return q.Count()
}

func (self *SHostwire) ValidateDeleteCondition(ctx context.Context) error {
	if self.GetGuestnicsCount() > 0 {
		return httperrors.NewNotEmptyError("guest on the host are using networks on this wire")
	}
	return self.SHostJointsBase.ValidateDeleteCondition(ctx)
}

func (self *SHostwire) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SHostwire) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

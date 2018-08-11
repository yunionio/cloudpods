package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SHostschedtagManager struct {
	SHostJointsManager
}

var HostschedtagManager *SHostschedtagManager

func init() {
	db.InitManager(func() {
		HostschedtagManager = &SHostschedtagManager{SHostJointsManager: NewHostJointsManager(SHostschedtag{}, "aggregate_hosts_tbl", "schedtaghost", "schedtaghosts", SchedtagManager)}
	})
}

type SHostschedtag struct {
	SHostJointsBase

	HostId     string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required" key_index:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
	SchedtagId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required" key_index:"true"` // =Column(VARCHAR(36, charset='ascii'), nullable=False)
}

func (joint *SHostschedtag) Master() db.IStandaloneModel {
	return db.JointMaster(joint)
}

func (joint *SHostschedtag) Slave() db.IStandaloneModel {
	return db.JointSlave(joint)
}

func (self *SHostschedtag) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SHostJointsBase.GetCustomizeColumns(ctx, userCred, query)
	return db.JointModelExtra(self, extra)
}

func (self *SHostschedtag) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SHostJointsBase.GetExtraDetails(ctx, userCred, query)
	return db.JointModelExtra(self, extra)
}

func (self *SHostschedtag) getHost() *SHost {
	obj, err := HostManager.FetchById(self.HostId)
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	return obj.(*SHost)
}

func (self *SHostschedtag) getSchedtag() *SSchedtag {
	obj, err := SchedtagManager.FetchById(self.SchedtagId)
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	return obj.(*SSchedtag)
}

func (self *SHostschedtag) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SHostschedtag) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

package models

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SHostschedtagManager struct {
	*SSchedtagJointsManager
}

var HostschedtagManager *SHostschedtagManager

func init() {
	db.InitManager(func() {
		HostschedtagManager = &SHostschedtagManager{
			SSchedtagJointsManager: NewSchedtagJointsManager(
				SHostschedtag{},
				"aggregate_hosts_tbl",
				"schedtaghost",
				"schedtaghosts",
				HostManager,
				SchedtagManager,
			),
		}
	})
}

type SHostschedtag struct {
	SSchedtagJointsBase

	HostId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
}

func (self *SHostschedtag) GetHost() *SHost {
	return self.Master().(*SHost)
}

func (self *SHostschedtag) GetHosts() ([]SHost, error) {
	hosts := []SHost{}
	err := self.GetSchedtag().GetObjects(&hosts)
	return hosts, err
}

func (self *SHostschedtag) Master() db.IStandaloneModel {
	return self.SSchedtagJointsBase.master(self)
}

func (self *SHostschedtag) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	return self.SSchedtagJointsBase.getCustomizeColumns(self, ctx, userCred, query)
}

func (self *SHostschedtag) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	return self.SSchedtagJointsBase.getExtraDetails(self, ctx, userCred, query)
}

func (self *SHostschedtag) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SSchedtagJointsBase.delete(self, ctx, userCred)
}

func (self *SHostschedtag) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SSchedtagJointsBase.detach(self, ctx, userCred)
}

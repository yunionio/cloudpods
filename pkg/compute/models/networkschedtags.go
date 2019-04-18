package models

import (
	"context"
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SNetworkschedtagManager struct {
	*SSchedtagJointsManager
}

var NetworkschedtagManager *SNetworkschedtagManager

func init() {
	db.InitManager(func() {
		NetworkschedtagManager = &SNetworkschedtagManager{
			SSchedtagJointsManager: NewSchedtagJointsManager(
				SNetworkschedtag{},
				"schedtag_networks_tbl",
				"schedtagnetwork",
				"schedtagnetworks",
				NetworkManager,
				SchedtagManager,
			),
		}
	})
}

type SNetworkschedtag struct {
	SSchedtagJointsBase

	NetworkId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
}

func (s *SNetworkschedtag) GetNetwork() *SNetwork {
	return s.Master().(*SNetwork)
}

func (s *SNetworkschedtag) GetNetworks() ([]SNetwork, error) {
	nets := []SNetwork{}
	err := s.GetSchedtag().GetObjects(&nets)
	return nets, err
}

func (s *SNetworkschedtag) Master() db.IStandaloneModel {
	return s.SSchedtagJointsBase.master(s)
}

func (s *SNetworkschedtag) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	return s.SSchedtagJointsBase.getCustomizeColumns(s, ctx, userCred, query)
}

func (s *SNetworkschedtag) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	return s.SSchedtagJointsBase.getExtraDetails(s, ctx, userCred, query)
}

func (s *SNetworkschedtag) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return s.SSchedtagJointsBase.delete(s, ctx, userCred)
}

func (s *SNetworkschedtag) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return s.SSchedtagJointsBase.detach(s, ctx, userCred)
}

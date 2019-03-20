package models

import (
	"context"
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SStorageschedtagManager struct {
	*SSchedtagJointsManager
}

var StorageschedtagManager *SStorageschedtagManager

func init() {
	db.InitManager(func() {
		StorageschedtagManager = &SStorageschedtagManager{
			SSchedtagJointsManager: NewSchedtagJointsManager(
				SStorageschedtag{},
				"schedtag_storages_tbl",
				"schedtagstorage",
				"schedtagstorages",
				StorageManager,
				SchedtagManager,
			),
		}
	})
}

type SStorageschedtag struct {
	SSchedtagJointsBase

	StorageId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
}

func (s *SStorageschedtag) GetStorage() *SStorage {
	return s.Master().(*SStorage)
}

func (s *SStorageschedtag) GetStorages() ([]SStorage, error) {
	storages := []SStorage{}
	err := s.GetSchedtag().GetObjects(&storages)
	return storages, err
}

func (joint *SStorageschedtag) Master() db.IStandaloneModel {
	return joint.SSchedtagJointsBase.master(joint)
}

func (joint *SStorageschedtag) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	return joint.SSchedtagJointsBase.getCustomizeColumns(joint, ctx, userCred, query)
}

func (joint *SStorageschedtag) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	return joint.SSchedtagJointsBase.getExtraDetails(joint, ctx, userCred, query)
}

func (joint *SStorageschedtag) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return joint.SSchedtagJointsBase.delete(joint, ctx, userCred)
}

func (joint *SStorageschedtag) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return joint.SSchedtagJointsBase.detach(joint, ctx, userCred)
}

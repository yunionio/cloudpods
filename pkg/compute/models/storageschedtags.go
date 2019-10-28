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
		StorageschedtagManager.SetVirtualObject(StorageschedtagManager)
	})
}

type SStorageschedtag struct {
	SSchedtagJointsBase

	StorageId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
}

func (manager *SStorageschedtagManager) GetSlaveFieldName() string {
	return "storage_id"
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

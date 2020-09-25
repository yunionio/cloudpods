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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SStorageschedtagManager struct {
	*SSchedtagJointsManager
	SStorageResourceBaseManager
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
			),
		}
		StorageschedtagManager.SetVirtualObject(StorageschedtagManager)
	})
}

type SStorageschedtag struct {
	SSchedtagJointsBase

	StorageId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
}

func (manager *SStorageschedtagManager) GetMasterFieldName() string {
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

func (joint *SStorageschedtag) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.StorageschedtagDetails, error) {
	return api.StorageschedtagDetails{}, nil
}

func (manager *SStorageschedtagManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.StorageschedtagDetails {
	rows := make([]api.StorageschedtagDetails, len(objs))

	schedRows := manager.SSchedtagJointsManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	storageIds := make([]string, len(rows))
	for i := range rows {
		rows[i] = api.StorageschedtagDetails{
			SchedtagJointResourceDetails: schedRows[i],
		}
		storageIds[i] = objs[i].(*SStorageschedtag).StorageId
	}

	storageIdMaps, err := db.FetchIdNameMap2(StorageManager, storageIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 hostIds fail %s", err)
		return rows
	}

	for i := range rows {
		if name, ok := storageIdMaps[storageIds[i]]; ok {
			rows[i].Storage = name
		}
	}

	return rows
}

func (joint *SStorageschedtag) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return joint.SSchedtagJointsBase.delete(joint, ctx, userCred)
}

func (joint *SStorageschedtag) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return joint.SSchedtagJointsBase.detach(joint, ctx, userCred)
}

func (manager *SStorageschedtagManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.StorageschedtagListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSchedtagJointsManager.ListItemFilter(ctx, q, userCred, query.SchedtagJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSchedtagJointsManager.ListItemFilter")
	}
	q, err = manager.SStorageResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StorageFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStorageResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *SStorageschedtagManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.StorageschedtagListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSchedtagJointsManager.OrderByExtraFields(ctx, q, userCred, query.SchedtagJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSchedtagJointsManager.OrderByExtraFields")
	}
	q, err = manager.SStorageResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StorageFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStorageResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

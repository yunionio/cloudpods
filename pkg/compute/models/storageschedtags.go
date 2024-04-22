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

	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	StorageschedtagManager *SStorageschedtagManager
	_                      ISchedtagJointModel = new(SStorageschedtag)
)

// +onecloud:swagger-gen-ignore
type SStorageschedtagManager struct {
	*SSchedtagJointsManager
	resourceBaseManager SStorageResourceBaseManager
}

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

// +onecloud:model-api-gen
type SStorageschedtag struct {
	SSchedtagJointsBase

	StorageId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
}

func (manager *SStorageschedtagManager) GetMasterFieldName() string {
	return "storage_id"
}

func (joint *SStorageschedtag) GetResourceId() string {
	return joint.StorageId
}

func (joint *SStorageschedtag) GetDetails(base api.SchedtagJointResourceDetails, resourceName string, isList bool) interface{} {
	out := api.StorageschedtagDetails{
		SchedtagJointResourceDetails: base,
	}
	out.Storage = resourceName
	return out
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
	q, err = manager.resourceBaseManager.ListItemFilter(ctx, q, userCred, query.StorageFilterListInput)
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
	q, err = manager.resourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StorageFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStorageResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

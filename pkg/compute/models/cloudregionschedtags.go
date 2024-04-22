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
	CloudregionschedtagManager *SCloudregionschedtagManager
	_                          ISchedtagJointModel = new(SCloudregionschedtag)
)

func init() {
	db.InitManager(func() {
		CloudregionschedtagManager = &SCloudregionschedtagManager{
			SSchedtagJointsManager: NewSchedtagJointsManager(
				SCloudregionschedtag{},
				"schedtag_cloudregions_tbl",
				"schedtagcloudregion",
				"schedtagcloudregions",
				CloudregionManager,
			),
		}
		CloudregionschedtagManager.SetVirtualObject(CloudregionschedtagManager)
	})
}

// +onecloud:swagger-gen-ignore
type SCloudregionschedtagManager struct {
	*SSchedtagJointsManager
	resourceBaseManager SCloudregionResourceBaseManager
}

// +onecloud:swagger-gen-ignore
type SCloudregionschedtag struct {
	SSchedtagJointsBase
	SCloudregionResourceBase
}

func (m *SCloudregionschedtagManager) GetMasterFieldName() string {
	return "cloudregion_id"
}

func (obj *SCloudregionschedtag) GetResourceId() string {
	return obj.CloudregionId
}

func (obj *SCloudregionschedtag) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return obj.SSchedtagJointsBase.delete(obj, ctx, userCred)
}

func (obj *SCloudregionschedtag) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return obj.SSchedtagJointsBase.detach(obj, ctx, userCred)
}

func (m *SCloudregionschedtagManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CloudregionschedtagListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = m.SSchedtagJointsManager.ListItemFilter(ctx, q, userCred, query.SchedtagJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSchedtagJointsManager.ListItemFilter")
	}
	q, err = m.resourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (obj *SCloudregionschedtag) GetDetails(base api.SchedtagJointResourceDetails, resourceName string, isList bool) interface{} {
	out := api.CloudregionschedtagDetails{
		SchedtagJointResourceDetails: base,
	}
	out.Cloudregion = resourceName
	return out
}

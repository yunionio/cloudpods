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
	ZoneschedtagManager *SZoneschedtagManager
	_                   ISchedtagJointModel = new(SZoneschedtag)
)

func init() {
	db.InitManager(func() {
		ZoneschedtagManager = &SZoneschedtagManager{
			SSchedtagJointsManager: NewSchedtagJointsManager(
				SZoneschedtag{},
				"schedtag_zones_tbl",
				"schedtagzone",
				"schedtagzones",
				ZoneManager,
			),
		}
		ZoneschedtagManager.SetVirtualObject(ZoneschedtagManager)
	})
}

// +onecloud:swagger-gen-ignore
type SZoneschedtagManager struct {
	*SSchedtagJointsManager
	resourceBaseManager SZoneResourceBaseManager
}

// +onecloud:swagger-gen-ignore
type SZoneschedtag struct {
	SSchedtagJointsBase
	SZoneResourceBase
}

func (m *SZoneschedtagManager) GetMasterFieldName() string {
	return "zone_id"
}

func (obj *SZoneschedtag) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return obj.SSchedtagJointsBase.delete(obj, ctx, userCred)
}

func (obj *SZoneschedtag) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return obj.SSchedtagJointsBase.detach(obj, ctx, userCred)
}

func (obj *SZoneschedtag) GetResourceId() string {
	return obj.ZoneId
}

func (m *SZoneschedtagManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ZoneschedtagListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = m.SSchedtagJointsManager.ListItemFilter(ctx, q, userCred, query.SchedtagJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSchedtagJointsManager.ListItemFilter")
	}
	q, err = m.resourceBaseManager.ListItemFilter(ctx, q, userCred, query.ZonalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (obj *SZoneschedtag) GetDetails(base api.SchedtagJointResourceDetails, resourceName string, isList bool) interface{} {
	out := api.ZoneschedtagDetails{
		SchedtagJointResourceDetails: base,
	}
	out.Zone = resourceName
	return out
}

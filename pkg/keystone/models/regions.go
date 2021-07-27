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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SRegionManager struct {
	db.SStandaloneResourceBaseManager
}

var RegionManager *SRegionManager

func init() {
	RegionManager = &SRegionManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SRegion{},
			"region",
			"region",
			"regions",
		),
	}
	RegionManager.SetVirtualObject(RegionManager)
}

/*
+------------------+--------------+------+-----+---------+-------+
| Field            | Type         | Null | Key | Default | Extra |
+------------------+--------------+------+-----+---------+-------+
| id               | varchar(255) | NO   | PRI | NULL    |       |
| description      | varchar(255) | NO   |     | NULL    |       |
| parent_region_id | varchar(255) | YES  |     | NULL    |       |
| extra            | text         | YES  |     | NULL    |       |
| created_at       | datetime     | YES  |     | NULL    |       |
+------------------+--------------+------+-----+---------+-------+
*/

type SRegion struct {
	db.SStandaloneResourceBase

	ParentRegionId string `width:"255" charset:"ascii" nulable:"true"`
	Extra          *jsonutils.JSONDict
}

func (manager *SRegionManager) InitializeData() error {
	q := manager.Query()
	q = q.IsNullOrEmpty("name")
	regions := make([]SRegion, 0)
	err := db.FetchModelObjects(manager, q, &regions)
	if err != nil {
		return err
	}
	for i := range regions {
		if gotypes.IsNil(regions[i].Extra) {
			continue
		}
		name, _ := regions[i].Extra.GetString("name")
		if len(name) == 0 {
			name = regions[i].Id
		}
		db.Update(&regions[i], func() error {
			regions[i].Name = name
			return nil
		})
	}
	return nil
}

func (region *SRegion) GetEndpointCount() (int, error) {
	q := EndpointManager.Query().Equals("region_id", region.Id)
	return q.CountWithError()
}

func (region *SRegion) ValidateDeleteCondition(ctx context.Context) error {
	epCnt, _ := region.GetEndpointCount()
	if epCnt > 0 {
		return httperrors.NewNotEmptyError("region contains endpoints")
	}
	return region.SStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (manager *SRegionManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.RegionDetails {
	rows := make([]api.RegionDetails, len(objs))
	stdRows := manager.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.RegionDetails{
			StandaloneResourceDetails: stdRows[i],
		}
		rows[i] = regionExtra(objs[i].(*SRegion), rows[i])
	}
	return rows
}

func regionExtra(region *SRegion, out api.RegionDetails) api.RegionDetails {
	out.EndpointCount, _ = region.GetEndpointCount()
	return out
}

func (manager *SRegionManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	idStr, _ := data.GetString("id")
	if len(idStr) == 0 {
		return nil, httperrors.NewInputParameterError("missing input field id")
	}
	if !data.Contains("name") {
		data.Set("name", jsonutils.NewString(idStr))
	}
	var err error
	input := apis.StandaloneResourceCreateInput{}
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal StandaloneResourceCreateInput fail %s", err)
	}
	input, err = manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))
	return data, nil
}

func (region *SRegion) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	err := region.SStandaloneResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
	if err != nil {
		return err
	}
	idStr, _ := data.GetString("id")
	region.Id = idStr
	return nil
}

// 区域列表
func (manager *SRegionManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.RegionListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SRegionManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.RegionListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SRegionManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

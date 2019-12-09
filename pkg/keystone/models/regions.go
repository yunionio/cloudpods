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

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
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

func (region *SRegion) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := region.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return regionExtra(region, extra)
}

func (region *SRegion) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := region.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return regionExtra(region, extra), nil
}

func regionExtra(region *SRegion, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	epCnt, _ := region.GetEndpointCount()
	extra.Add(jsonutils.NewInt(int64(epCnt)), "endpoint_count")
	return extra
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

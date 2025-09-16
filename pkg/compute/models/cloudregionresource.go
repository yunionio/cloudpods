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
	"database/sql"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SCloudregionResourceBase struct {
	// 归属区域ID
	CloudregionId string `width:"36" charset:"ascii" nullable:"false" list:"user" default:"default" create:"optional" json:"cloudregion_id"`
}

type SCloudregionResourceBaseManager struct{}

func ValidateCloudregionResourceInput(ctx context.Context, userCred mcclient.TokenCredential, input api.CloudregionResourceInput) (*SCloudregion, api.CloudregionResourceInput, error) {
	regionObj, err := CloudregionManager.FetchByIdOrName(ctx, userCred, input.CloudregionId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, input, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", CloudregionManager.Keyword(), input.CloudregionId)
		} else {
			return nil, input, errors.Wrap(err, "CloudregionManager.FetchByIdOrName")
		}
	}
	input.CloudregionId = regionObj.GetId()
	return regionObj.(*SCloudregion), input, nil
}

func ValidateCloudregionId(ctx context.Context, userCred mcclient.TokenCredential, regionId string) (*SCloudregion, error) {
	regionObj, err := CloudregionManager.FetchByIdOrName(ctx, userCred, regionId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", CloudregionManager.Keyword(), regionId)
		} else {
			return nil, errors.Wrap(err, "CloudregionManager.FetchByIdOrName")
		}
	}
	return regionObj.(*SCloudregion), nil
}

func (self *SCloudregionResourceBase) GetRegion() (*SCloudregion, error) {
	region, err := CloudregionManager.FetchById(self.CloudregionId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegion(%s)", self.CloudregionId)
	}
	return region.(*SCloudregion), nil
}

func (self *SCloudregionResourceBase) GetZoneBySuffix(suffix string) (*SZone, error) {
	sq := ZoneManager.Query().SubQuery()
	q := sq.Query().Filter(
		sqlchemy.AND(
			sqlchemy.Equals(sq.Field("cloudregion_id"), self.CloudregionId),
			sqlchemy.Endswith(sq.Field("external_id"), suffix),
		),
	)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, suffix)
	}
	if count > 1 {
		return nil, errors.Wrapf(cloudprovider.ErrDuplicateId, suffix)
	}
	zone := &SZone{}
	zone.SetModelManager(ZoneManager, zone)
	return zone, q.First(zone)
}

func (manager *SCloudregionResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CloudregionResourceInfo {
	rows := make([]api.CloudregionResourceInfo, len(objs))
	regionIds := make([]string, len(objs))
	for i := range objs {
		var base *SCloudregionResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SCloudregionResourceBase in %#v: %s", objs[i], err)
		} else if base != nil && len(base.CloudregionId) > 0 {
			regionIds[i] = base.CloudregionId
		}
	}
	regions := make(map[string]SCloudregion)
	err := db.FetchStandaloneObjectsByIds(CloudregionManager, regionIds, regions)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}
	for i := range rows {
		if region, ok := regions[regionIds[i]]; ok {
			rows[i] = region.GetRegionInfo(ctx)
		} else {
			rows[i] = api.CloudregionResourceInfo{}
		}
	}
	return rows
}

func (manager *SCloudregionResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.RegionalFilterListInput,
) (*sqlchemy.SQuery, error) {
	return managedResourceFilterByRegion(ctx, q, query, "", nil)
}

func (manager *SCloudregionResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.RegionalFilterListInput,
) (*sqlchemy.SQuery, error) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, nil
	}
	orderQ := CloudregionManager.Query("id")
	orderSubQ := orderQ.SubQuery()
	orderQ, orders, fields := manager.GetOrderBySubQuery(orderQ, orderSubQ, orderQ.Field("id"), userCred, query, nil, nil)
	q = q.LeftJoin(orderSubQ, sqlchemy.Equals(q.Field("cloudregion_id"), orderSubQ.Field("id")))
	q = db.OrderByFields(q, orders, fields)
	return q, nil
}

func (manager *SCloudregionResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	if field == "region" {
		regionQuery := CloudregionManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(regionQuery.Field("name", field))
		q = q.Join(regionQuery, sqlchemy.Equals(q.Field("cloudregion_id"), regionQuery.Field("id")))
		q.GroupBy(regionQuery.Field("name"))
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SCloudregionResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	subq *sqlchemy.SSubQuery,
	subqField sqlchemy.IQueryField,
	userCred mcclient.TokenCredential,
	query api.RegionalFilterListInput,
	orders []string,
	fields []sqlchemy.IQueryField,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	if !db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		return q, orders, fields
	}
	regions := CloudregionManager.Query().SubQuery()
	q = q.LeftJoin(regions, sqlchemy.Equals(subqField, regions.Field("id")))
	q = q.AppendField(regions.Field("name").Label("region"))
	q = q.AppendField(regions.Field("city"))
	orders = append(orders, query.OrderByRegion, query.OrderByCity)
	fields = append(fields, subq.Field("region"), subq.Field("city"))
	return q, orders, fields
}

func (manager *SCloudregionResourceBaseManager) GetOrderByFields(query api.RegionalFilterListInput) []string {
	return []string{query.OrderByRegion, query.OrderByCity}
}

func (manager *SCloudregionResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		regionsQ := CloudregionManager.Query("id", "name").SubQuery()
		q = q.LeftJoin(regionsQ, sqlchemy.Equals(q.Field("cloudregion_id"), regionsQ.Field("id")))
		if keys.Contains("region") {
			q = q.AppendField(regionsQ.Field("name", "region"))
		}
	}
	return q, nil
}

func (manager *SCloudregionResourceBaseManager) GetExportKeys() []string {
	return []string{"region"}
}

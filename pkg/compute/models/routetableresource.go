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

	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SRouteTableResourceBase struct {
	RouteTableId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" json:"route_table_id"`
}

type SRouteTableResourceBaseManager struct {
	SVpcResourceBaseManager
}

func (manager *SRouteTableResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.RouteTableFilterList,
) (*sqlchemy.SQuery, error) {
	if len(query.RouteTableId) > 0 {
		routeTable, err := RouteTableManager.FetchByIdOrName(ctx, userCred, query.RouteTableId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("route_table", query.RouteTableId)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		q = q.Equals("route_table_id", routeTable.GetId())
	}
	subq := RouteTableManager.Query("id").Snapshot()
	subq, err := manager.SVpcResourceBaseManager.ListItemFilter(ctx, subq, userCred, query.VpcFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemFilter")
	}
	if subq.IsAltered() {
		q = q.Filter(sqlchemy.In(q.Field("route_table_id"), subq.SubQuery()))
	}
	return q, nil
}

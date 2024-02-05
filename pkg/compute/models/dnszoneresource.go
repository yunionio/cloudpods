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
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SDnsZoneResourceBase struct {
	DnsZoneId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" json:"dns_zone_id"`
}

type SDnsZoneResourceBaseManager struct {
	SManagedResourceBaseManager
}

func (manager *SDnsZoneResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DnsZoneFilterListBase,
) (*sqlchemy.SQuery, error) {
	if len(query.DnsZoneId) > 0 {
		_, err := validators.ValidateModel(ctx, userCred, DnsZoneManager, &query.DnsZoneId)
		if err != nil {
			return nil, err
		}
		q = q.Equals("dns_zone_id", query.DnsZoneId)
	}

	subq := DnsZoneManager.Query("id").Snapshot()
	subq, err := manager.SManagedResourceBaseManager.ListItemFilter(ctx, subq, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	if subq.IsAltered() {
		q = q.Filter(sqlchemy.In(q.Field("dns_zone_id"), subq.SubQuery()))
	}

	return q, nil
}

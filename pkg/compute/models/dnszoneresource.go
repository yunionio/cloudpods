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

type SDnsZoneResourceBase struct {
	DnsZoneId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" json:"dns_zone_id"`
}

type SDnsZoneResourceBaseManager struct{}

func (manager *SDnsZoneResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DnsZoneFilterListBase,
) (*sqlchemy.SQuery, error) {
	if len(query.DnsZoneId) > 0 {
		dnsZone, err := DnsZoneManager.FetchByIdOrName(userCred, query.DnsZoneId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("dns_zone", query.DnsZoneId)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		q = q.Equals("dns_zone_id", dnsZone.GetId())
	}
	return q, nil
}

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

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SServiceResourceBaseManager struct{}

func (manager *SServiceResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ServiceFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.ServiceId) > 0 {
		serviceObj, err := ServiceManager.FetchByIdOrName(ctx, userCred, query.ServiceId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(ServiceManager.Keyword(), query.ServiceId)
			} else {
				return nil, errors.Wrap(err, "ServiceManager.FetchByIdOrName")
			}
		}
		q = q.Equals("service_id", serviceObj.GetId())
	}
	if len(query.ServiceType) > 0 {
		subq := ServiceManager.Query("id").Equals("type", query.ServiceType).SubQuery()
		q = q.In("service_id", subq)
	}
	return q, nil
}

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

type SInterVpcNetworkResourceBase struct {
	InterVpcNetworkId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" json:"inter_vpc_network_id"`
}

type SInterVpcNetworkResourceBaseManager struct{}

func (manager *SInterVpcNetworkResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.InterVpcNetworkFilterListBase,
) (*sqlchemy.SQuery, error) {
	if len(query.InterVpcNetworkId) > 0 {
		network, err := InterVpcNetworkManager.FetchByIdOrName(ctx, userCred, query.InterVpcNetworkId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("inter_vpc_network", query.InterVpcNetworkId)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		q = q.Equals("inter_vpc_network_id", network.GetId())
	}
	return q, nil
}

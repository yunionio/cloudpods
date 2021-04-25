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
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/cloudproxy"
	cloudproxy_api "yunion.io/x/onecloud/pkg/apis/cloudproxy"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SProxyMatch struct {
	db.SVirtualResourceBase

	ProxyEndpointId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	MatchScope      string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	MatchValue      string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
}

type SProxyMatchManager struct {
	db.SVirtualResourceBaseManager
}

var ProxyMatchManager *SProxyMatchManager

func init() {
	ProxyMatchManager = &SProxyMatchManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SProxyMatch{},
			"proxy_matches_tbl",
			"proxy_match",
			"proxy_matches",
		),
	}
	ProxyMatchManager.SetVirtualObject(ProxyMatchManager)
}

func (man *SProxyMatchManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	matchScopeV := validators.NewStringChoicesValidator("match_scope", api.PM_SCOPES)
	endpointV := validators.NewModelIdOrNameValidator("proxy_endpoint", ProxyEndpointManager.Keyword(), ownerId)
	for _, v := range []validators.IValidator{
		matchScopeV,
		endpointV,
	} {
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (pm *SProxyMatch) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	matchScopeV := validators.NewStringChoicesValidator("match_scope", api.PM_SCOPES)
	endpointV := validators.NewModelIdOrNameValidator("proxy_endpoint", ProxyEndpointManager.Keyword(), userCred)
	for _, v := range []validators.IValidator{
		matchScopeV,
		endpointV,
	} {
		v.Optional(true)
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (man *SProxyMatchManager) findMatch(ctx context.Context, networkId, vpcId string) *SProxyMatch {
	q := man.Query()
	qfScope := q.Field("match_scope")
	qfValue := q.Field("match_value")
	q = q.Filter(
		sqlchemy.OR(
			sqlchemy.AND(
				sqlchemy.Equals(qfScope, api.PM_SCOPE_VPC),
				sqlchemy.Equals(qfValue, vpcId),
			),
			sqlchemy.AND(
				sqlchemy.Equals(qfScope, api.PM_SCOPE_NETWORK),
				sqlchemy.Equals(qfValue, networkId),
			),
		),
	)

	var pms []SProxyMatch
	if err := db.FetchModelObjects(man, q, &pms); err != nil {
		return nil
	}
	var r *SProxyMatch
	for _, pm := range pms {
		if pm.MatchScope == api.PM_SCOPE_NETWORK {
			return &pm
		}
		r = &pm
	}
	return r
}

func (man *SProxyMatchManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input cloudproxy_api.ProxyMatchListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}

	if len(input.ProxyEndpointId) > 0 {
		_, err := validators.ValidateModel(userCred, ProxyEndpointManager, &input.ProxyEndpointId)
		if err != nil {
			return nil, err
		}

		q = q.Equals("proxy_endpoint_id", input.ProxyEndpointId)
	}

	return q, nil
}

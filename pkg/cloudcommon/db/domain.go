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

package db

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SDomainizedResourceBaseManager struct {
}

type SDomainizedResourceBase struct {
	DomainId string `width:"64" charset:"ascii" default:"default" nullable:"false" index:"true" list:"user"`
}

func (manager *SDomainizedResourceBaseManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeDomain
}

func (manager *SDomainizedResourceBaseManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		switch scope {
		case rbacutils.ScopeDomain:
			q = q.Equals("domain_id", owner.GetProjectDomainId())
		}
	}
	return q
}

func (manager *SDomainizedResourceBaseManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return FetchDomainInfo(ctx, data)
}

func (model *SDomainizedResourceBase) GetOwnerId() mcclient.IIdentityProvider {
	owner := SOwnerId{DomainId: model.DomainId}
	return &owner
}

func ValidateCreateDomainId(domainId string) error {
	if !consts.GetNonDefaultDomainProjects() && domainId != identity.DEFAULT_DOMAIN_ID {
		return httperrors.NewForbiddenError("project in non-default domain is prohibited")
	}
	return nil
}

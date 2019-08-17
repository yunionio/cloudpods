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

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SProjectizedResourceBaseManager struct {
}

type SProjectizedResourceBase struct {
	SDomainizedResourceBase

	ProjectId string `name:"tenant_id" width:"128" charset:"ascii" nullable:"false" index:"true" list:"user"`
}

func (model *SProjectizedResourceBase) GetOwnerId() mcclient.IIdentityProvider {
	owner := SOwnerId{DomainId: model.DomainId, ProjectId: model.ProjectId}
	return &owner
}

func (manager *SProjectizedResourceBaseManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		switch scope {
		case rbacutils.ScopeProject:
			q = q.Equals("tenant_id", owner.GetProjectId())
		case rbacutils.ScopeDomain:
			q = q.Equals("domain_id", owner.GetProjectDomainId())
		}
		/*if len(owner.GetProjectId()) > 0 {
			q = q.Equals("tenant_id", owner.GetProjectId())
		} else if len(owner.GetProjectDomainId()) > 0 {
			q = q.Equals("domain_id", owner.GetProjectDomainId())
		}*/
	}
	return q
}

func (manager *SProjectizedResourceBaseManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeProject
}

func (manager *SProjectizedResourceBaseManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return FetchProjectInfo(ctx, data)
}

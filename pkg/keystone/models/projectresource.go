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
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// +onecloud:swagger-gen-ignore
type SProjectResourceBaseManager struct{}

func (manager *SProjectResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ProjectFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.ProjectId) > 0 {
		var ownerId mcclient.IIdentityProvider
		if len(query.ProjectDomainId) > 0 {
			domain, err := DomainManager.FetchDomainByIdOrName(query.ProjectDomainId)
			if err != nil {
				if errors.Cause(err) == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError2(DomainManager.Keyword(), query.ProjectDomainId)
				} else {
					return nil, errors.Wrap(err, "DomainManager.FetchDomainByIdOrName")
				}
			}
			ownerId = &db.SOwnerId{
				Domain:   domain.Name,
				DomainId: domain.Id,
			}

		} else {
			ownerId = userCred
		}
		projObj, err := ProjectManager.FetchByIdOrName(ctx, ownerId, query.ProjectId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(ProjectManager.Keyword(), query.ProjectId)
			} else {
				return nil, errors.Wrap(err, "ProjectManager.FetchByIdOrName")
			}
		}
		q = q.Equals("project_id", projObj.GetId())
	}
	return q, nil
}

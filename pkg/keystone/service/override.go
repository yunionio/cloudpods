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

package service

import (
	"context"
	"database/sql"

	"github.com/golang-plus/uuid"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/models"
)

func keystoneUUIDGenerator() string {
	id, _ := uuid.NewV4()
	return id.Format(uuid.StyleWithoutDash)
}

func keystoneProjectFetcher(ctx context.Context, idstr string, domainId string) (*db.STenant, error) {
	tenantObj, err := models.ProjectManager.FetchProject(idstr, idstr, domainId, domainId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "tenant %s", idstr)
		} else {
			return nil, errors.Wrap(err, "models.ProjectManager.FetchByIdOrName")
		}
	}
	ret := project2Tenant(tenantObj)
	return &ret, nil
}

func keystoneDomainFetcher(ctx context.Context, idstr string) (*db.STenant, error) {
	domainObj, err := models.DomainManager.FetchByIdOrName(ctx, nil, idstr)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "domain %s", idstr)
		} else {
			return nil, errors.Wrap(err, "models.DomainManager.FetchByIdOrName")
		}
	}
	ret := domain2Tenant(domainObj.(*models.SDomain))
	return &ret, nil
}

func project2Tenant(tenant *models.SProject) db.STenant {
	ret := db.STenant{}
	ret.Id = tenant.Id
	ret.Name = tenant.Name
	ret.DomainId = tenant.DomainId
	ret.Domain = tenant.GetDomain().Name
	return ret
}

func domain2Tenant(domain *models.SDomain) db.STenant {
	ret := db.STenant{}
	ret.Id = domain.Id
	ret.Name = domain.Name
	ret.DomainId = api.KeystoneDomainRoot
	ret.Domain = api.KeystoneDomainRoot
	return ret
}

func keystoneProjectsFetcher(ctx context.Context, idList []string, isDomain bool) map[string]db.STenant {
	if isDomain {
		domains := make(map[string]models.SDomain)
		err := db.FetchStandaloneObjectsByIds(models.DomainManager, idList, &domains)
		if err != nil {
			log.Errorf("FetchStandaloneObjectsByIds for domain fail %s", err)
			return nil
		}
		ret := make(map[string]db.STenant)
		for id, domain := range domains {
			ret[id] = domain2Tenant(&domain)
		}
		return ret
	} else {
		projects := make(map[string]models.SProject)
		err := db.FetchStandaloneObjectsByIds(models.ProjectManager, idList, &projects)
		if err != nil {
			log.Errorf("FetchStandaloneObjectsByIds for project fail %s", err)
			return nil
		}
		ret := make(map[string]db.STenant)
		for id, project := range projects {
			ret[id] = project2Tenant(&project)
		}
		return ret
	}
}

func keystoneProjectQuery(fields ...string) *sqlchemy.SQuery {
	return models.ProjectManager.Query(fields...)
}

func keystoneDomainQuery(fields ...string) *sqlchemy.SQuery {
	return models.DomainManager.Query(fields...)
}

func keystoneUserFetcher(ctx context.Context, idstr string) (*db.SUser, error) {
	userObj, err := models.UserManager.FetchByIdOrName(ctx, nil, idstr)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "user %s", idstr)
		} else {
			return nil, errors.Wrap(err, "models.UserManager.FetchByIdOrName")
		}
	}
	ret := user2User(userObj.(*models.SUser))
	return &ret, nil
}

func user2User(u *models.SUser) db.SUser {
	ret := db.SUser{}
	ret.Id = u.Id
	ret.Name = u.Name
	ret.DomainId = u.DomainId
	ret.Domain = u.GetDomain().Name
	return ret
}

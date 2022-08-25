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

	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
)

func FetchAllRemoteDomainProjects(ctx context.Context) ([]*db.STenant, []*db.STenant, error) {
	s := auth.GetAdminSession(ctx, consts.GetRegion())
	projects := make([]*db.STenant, 0)
	domains := make([]*db.STenant, 0)
	var count int
	domainMap := make(map[string]string, 0)
	for {
		listParam := jsonutils.NewDict()
		listParam.Add(jsonutils.NewString("system"), "scope")
		listParam.Add(jsonutils.NewInt(0), "limit")
		listParam.Add(jsonutils.NewInt(int64(count)), "offset")
		listParam.Add(jsonutils.JSONTrue, "details")
		result, err := identity.Projects.List(s, listParam)
		if err != nil {
			return domains, projects, errors.Wrap(err, "list projects from keystone")
		}
		for _, data := range result.Data {
			item := db.SCachedTenant{}
			data.Unmarshal(&item)
			project, err := db.TenantCacheManager.Save(ctx, item, true)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "save project %s to cache", data.String())
			}
			projects = append(projects, project)
			domainMap[item.DomainId] = item.ProjectDomain
		}
		total := result.Total
		count = count + len(result.Data)
		if count >= total {
			break
		}
	}
	for domainId, domainName := range domainMap {
		item := db.SCachedTenant{
			Id:            domainId,
			Name:          domainName,
			DomainId:      identityapi.KeystoneDomainRoot,
			ProjectDomain: identityapi.KeystoneDomainRoot,
		}
		domain, err := db.TenantCacheManager.Save(ctx, item, false)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "save domain %s:%s to cache", domainId, domainName)
		}
		domains = append(domains, domain)
	}
	return domains, projects, nil
}

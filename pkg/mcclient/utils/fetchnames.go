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

package utils

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/rbacscope"

	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
)

func MapKeys(idMap map[string]string) []string {
	keys := make([]string, len(idMap))
	idx := 0
	for k := range idMap {
		keys[idx] = k
		idx += 1
	}
	return keys
}

func FetchDomainNames(ctx context.Context, domainMap map[string]string) (map[string]string, error) {
	s := auth.GetAdminSession(ctx, consts.GetRegion())
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString(fmt.Sprintf("id.equals(%s)", strings.Join(MapKeys(domainMap), ","))), "filter.0")
	query.Add(jsonutils.NewInt(int64(len(domainMap))), "limit")
	query.Add(jsonutils.NewString(string(rbacscope.ScopeSystem)), "scope")
	query.Add(jsonutils.JSONTrue, "details")
	results, err := modules.Domains.List(s, query)
	if err == nil {
		for i := range results.Data {
			// update domain cache
			item := db.SCachedTenant{}
			results.Data[i].Unmarshal(&item)
			item.ProjectDomain = identityapi.KeystoneDomainRoot
			item.DomainId = identityapi.KeystoneDomainRoot
			db.TenantCacheManager.Save(ctx, item, true)
			domainMap[item.Id] = item.Name
		}
		for k, v := range domainMap {
			if len(v) == 0 {
				db.TenantCacheManager.Delete(ctx, k)
			}
		}
		return domainMap, nil
	} else {
		return domainMap, err
	}
}

func FetchTenantNames(ctx context.Context, tenantMap map[string]string) (map[string]string, error) {
	s := auth.GetAdminSession(ctx, consts.GetRegion())
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString(fmt.Sprintf("id.equals(%s)", strings.Join(MapKeys(tenantMap), ","))), "filter.0")
	query.Add(jsonutils.NewInt(int64(len(tenantMap))), "limit")
	query.Add(jsonutils.NewString(string(rbacscope.ScopeSystem)), "scope")
	query.Add(jsonutils.JSONTrue, "details")
	results, err := modules.Projects.List(s, query)
	if err == nil {
		for i := range results.Data {
			// update project cache
			item := db.SCachedTenant{}
			results.Data[i].Unmarshal(&item)
			db.TenantCacheManager.Save(ctx, item, true)
			tenantMap[item.Id] = item.Name
		}
		for k, v := range tenantMap {
			if len(v) == 0 {
				db.TenantCacheManager.Delete(ctx, k)
			}
		}
		return tenantMap, nil
	} else {
		return tenantMap, err
	}
}

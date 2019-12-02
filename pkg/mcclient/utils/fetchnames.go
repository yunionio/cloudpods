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
	"strings"
	"fmt"

	"yunion.io/x/jsonutils"

	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
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
	s := auth.GetAdminSession(ctx, consts.GetRegion(), "v1")
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString(fmt.Sprintf("id.equals(%s)", strings.Join(MapKeys(domainMap), ","))), "filter.0")
	query.Add(jsonutils.NewInt(int64(len(domainMap))), "limit")
	results, err := modules.Domains.List(s, query)
	if err == nil {
		for i := range results.Data {
			// update cache
			domainId, _ := results.Data[i].GetString("id")
			domainName, _ := results.Data[i].GetString("name")
			db.TenantCacheManager.Save(ctx, domainId, domainName, identityapi.KeystoneDomainRoot, identityapi.KeystoneDomainRoot)
			domainMap[domainId] = domainName
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
	s := auth.GetAdminSession(ctx, consts.GetRegion(), "v1")
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString(fmt.Sprintf("id.equals(%s)", strings.Join(MapKeys(tenantMap), ","))), "filter.0")
	query.Add(jsonutils.NewInt(int64(len(tenantMap))), "limit")
	results, err := modules.Projects.List(s, query)
	if err == nil {
		for i := range results.Data {
			// update cache
			projId, _ := results.Data[i].GetString("id")
			projName, _ := results.Data[i].GetString("name")
			projDomainId, _ := results.Data[i].GetString("domain_id")
			projDomain, _ := results.Data[i].GetString("project_domain")
			db.TenantCacheManager.Save(ctx, projId, projName, projDomainId, projDomain)
			tenantMap[projId] = projName
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

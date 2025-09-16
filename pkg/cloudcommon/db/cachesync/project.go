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

package cachesync

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
)

type tenantCacheSyncWorker struct {
	ids []string
}

func (w *tenantCacheSyncWorker) Run() {
	log.Debugf("[tenantCacheSyncWorker] Run project cache sync worker ...")
	err := syncProjects(context.Background(), w.ids)
	if err != nil {
		log.Errorf("fail to syncProjects %s", err)
	}
}

func (w *tenantCacheSyncWorker) Dump() string {
	return "tenantCacheSyncWorker"
}

func syncProjects(ctx context.Context, ids []string) error {
	s := auth.GetAdminSession(ctx, consts.GetRegion())
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewInt(1024), "limit")
	query.Add(jsonutils.NewString(string(rbacscope.ScopeSystem)), "scope")
	query.Add(jsonutils.JSONTrue, "details")
	query.Add(jsonutils.NewString("all"), "pending_delete")
	query.Add(jsonutils.NewString("all"), "delete")
	if len(ids) > 0 {
		log.Debugf("to syncProjects for %s", jsonutils.Marshal(ids))
		query.Add(jsonutils.NewStringArray(ids), "id")
	}
	total := -1
	offset := 0
	for total < 0 || offset < total {
		query.Set("offset", jsonutils.NewInt(int64(offset)))
		results, err := modules.Projects.List(s, query)
		if err != nil {
			return errors.Wrap(err, "Projects.List")
		}
		total = results.Total
		for i := range results.Data {
			// update project cache
			item := db.SCachedTenant{}
			deleted := jsonutils.QueryBoolean(results.Data[i], "deleted", false)
			err := results.Data[i].Unmarshal(&item)
			if err == nil && !deleted {
				db.TenantCacheManager.Save(ctx, item, true)
			} else if deleted {
				tenantObj, _ := db.TenantCacheManager.FetchById(item.Id)
				if tenantObj != nil {
					tenantObj.Delete(ctx, nil)
				}
			}
			offset++
		}
	}
	return nil
}

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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"

	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
)

var (
	tenantCacheSyncWorkerMan = appsrv.NewWorkerManagerIgnoreOverflow("tenant_cache_sync_worker", 1, 1, true, true)
)

func StartTenantCacheSync(ctx context.Context, intvalSeconds int) {

	go runTenantCacheSync(ctx, intvalSeconds)
}

func runTenantCacheSync(ctx context.Context, intvalSeconds int) {
	for {
		select {
		case <-time.After(time.Duration(intvalSeconds) * time.Second):
			tenantCacheSyncWorkerMan.Run(&tenantCacheSyncWorker{ctx}, nil, nil)
		}
	}
}

type tenantCacheSyncWorker struct {
	ctx context.Context
}

func (w *tenantCacheSyncWorker) Run() {
	log.Debugf("Run project and domain cache sync worker ...")
	err := syncDomains(w.ctx)
	if err != nil {
		log.Errorf("fail to syncDomains %s", err)
	}
	err = syncProjects(w.ctx)
	if err != nil {
		log.Errorf("fail to syncProjects %s", err)
	}
}

func (w *tenantCacheSyncWorker) Dump() string {
	return "tenantCacheSyncWorker"
}

func syncDomains(ctx context.Context) error {
	s := auth.GetAdminSession(ctx, consts.GetRegion())
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewInt(1024), "limit")
	query.Add(jsonutils.NewString(string(rbacscope.ScopeSystem)), "scope")
	query.Add(jsonutils.JSONTrue, "details")
	total := -1
	offset := 0
	for total < 0 || offset < total {
		query.Set("offset", jsonutils.NewInt(int64(offset)))
		results, err := modules.Domains.List(s, query)
		if err != nil {
			log.Errorf("syncDomain error %s", err)
			return errors.Wrap(err, "Domains.List")
		}
		total = results.Total
		for i := range results.Data {
			// update domain cache
			item := SCachedTenant{}
			results.Data[i].Unmarshal(&item)
			item.ProjectDomain = identityapi.KeystoneDomainRoot
			item.DomainId = identityapi.KeystoneDomainRoot
			TenantCacheManager.Save(ctx, item, true)
			offset++
		}
	}
	return nil
}

func syncProjects(ctx context.Context) error {
	s := auth.GetAdminSession(ctx, consts.GetRegion())
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewInt(1024), "limit")
	query.Add(jsonutils.NewString(string(rbacscope.ScopeSystem)), "scope")
	query.Add(jsonutils.JSONTrue, "details")
	total := -1
	offset := 0
	for total < 0 || offset < total {
		query.Set("offset", jsonutils.NewInt(int64(offset)))
		results, err := modules.Projects.List(s, query)
		if err != nil {
			log.Errorf("syncProjects error %s", err)
			return errors.Wrap(err, "Projects.List")
		}
		total = results.Total
		for i := range results.Data {
			// update project cache
			item := SCachedTenant{}
			results.Data[i].Unmarshal(&item)
			TenantCacheManager.Save(ctx, item, true)
			offset++
		}
	}
	return nil
}

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

package quotas

import (
	"context"
	"database/sql"
	"strings"
	"sync"

	"yunion.io/x/log"

	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

var (
	usageCalculateWorker = appsrv.NewWorkerManager("usageCalculateWorker", 1, 1024, true)
	usageDirtyMap        = make(map[string]bool, 0)
	usageDirtyMapLock    = &sync.Mutex{}
)

type sUsageCalculateJob struct {
	manager  *SQuotaBaseManager
	scope    rbacutils.TRbacScope
	ownerId  mcclient.IIdentityProvider
	platform []string
}

func setDirty(key string) {
	usageDirtyMapLock.Lock()
	defer usageDirtyMapLock.Unlock()

	usageDirtyMap[key] = true
}

func clearDirty(key string) {
	usageDirtyMapLock.Lock()
	defer usageDirtyMapLock.Unlock()

	delete(usageDirtyMap, key)
}

func isDirty(key string) bool {
	usageDirtyMapLock.Lock()
	defer usageDirtyMapLock.Unlock()

	if _, ok := usageDirtyMap[key]; ok {
		return true
	}
	return false
}

func (manager *SQuotaBaseManager) PostUsageJob(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, platform []string, usageChan chan IQuota, cleanEmpty bool) {
	key := getMemoryStoreKey(scope, ownerId, platform)
	setDirty(key)

	usageCalculateWorker.Run(func() {
		ctx := context.Background()

		if !isDirty(key) {
			return
		}
		clearDirty(key)

		usage := manager.newQuota()
		err := usage.FetchUsage(ctx, scope, ownerId, platform)
		if err != nil {
			return
		}

		var save bool
		if usage.IsEmpty() && cleanEmpty {
			// check existence of project
			s := auth.GetAdminSession(ctx, consts.GetRegion(), "v1")
			if scope == rbacutils.ScopeDomain {
				domain, err := modules.Domains.GetById(s, ownerId.GetProjectDomainId(), nil)
				if err == nil {
					// update cache
					domainId, _ := domain.GetString("id")
					domainName, _ := domain.GetString("name")
					db.TenantCacheManager.Save(ctx, domainId, domainName, identityapi.KeystoneDomainRoot, identityapi.KeystoneDomainRoot)
					save = true
				} else if httputils.ErrorCode(err) == 404 {
					// remove cache and quota
					db.TenantCacheManager.Delete(ctx, ownerId.GetProjectDomainId())
					save = false
				}
			} else {
				proj, err := modules.Projects.GetById(s, ownerId.GetProjectId(), nil)
				if err == nil {
					// update cache
					projId, _ := proj.GetString("id")
					projName, _ := proj.GetString("name")
					projDomainId, _ := proj.GetString("domain_id")
					projDomain, _ := proj.GetString("project_domain")
					db.TenantCacheManager.Save(ctx, projId, projName, projDomainId, projDomain)
					save = true
				} else if httputils.ErrorCode(err) == 404 {
					// remove cache and quota
					db.TenantCacheManager.Delete(ctx, ownerId.GetProjectId())
					save = false
				}
			}
		} else {
			save = true
		}
		if save {
			manager.usageStore.SetQuota(ctx, nil, scope, ownerId, platform, usage)
		} else {
			manager.usageStore.DeleteQuota(ctx, nil, scope, ownerId, platform)
			manager.DeleteQuota(ctx, nil, scope, ownerId, platform)
		}

		clearDirty(key)

		if usageChan != nil {
			usageChan <- usage
		}
	}, nil, nil)
}

func (manager *SQuotaBaseManager) CalculateQuotaUsages(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	log.Infof("CalculateQuotaUsages")
	rows, err := manager.Query("domain_id", "tenant_id", "platform").IsNullOrEmpty("platform").Rows()
	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("query quotas fail %s", err)
		}
		return
	}
	defer rows.Close()

	for rows.Next() {
		var domainId, projectId, platform string
		err := rows.Scan(&domainId, &projectId, &platform)
		if err != nil {
			log.Errorf("scan domain_id, project_id, platform error %s", err)
			return
		}
		scope := rbacutils.ScopeProject
		owner := db.SOwnerId{
			DomainId:  domainId,
			ProjectId: projectId,
		}
		if len(projectId) == 0 {
			scope = rbacutils.ScopeDomain
		}
		platforms := strings.Split(platform, nameSeparator)
		// log.Debugf("PostUsageJob %s %s %s", scope, owner, platforms)
		manager.PostUsageJob(scope, &owner, platforms, nil, true)
	}
}

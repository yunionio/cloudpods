package quotas

import (
	"context"
	"sync"

	"database/sql"
	"strings"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
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

func (manager *SQuotaBaseManager) PostUsageJob(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, platform []string, usageChan chan IQuota) {
	key := getMemoryStoreKey(scope, ownerId, platform)
	setDirty(key)

	usageCalculateWorker.Run(func() {
		if !isDirty(key) {
			return
		}
		clearDirty(key)

		usage := manager.newQuota()
		err := usage.FetchUsage(context.Background(), scope, ownerId, platform)
		if err != nil {
			return
		}

		manager.usageStore.SetQuota(context.Background(), nil, scope, ownerId, platform, usage)

		clearDirty(key)

		if usageChan != nil {
			usageChan <- usage
		}
	}, nil, nil)
}

func (manager *SQuotaBaseManager) CalculateQuotaUsages(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	rows, err := manager.Query("domain_id", "tenant_id", "platform").Rows()
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
		manager.PostUsageJob(scope, &owner, platforms, nil)
	}
}

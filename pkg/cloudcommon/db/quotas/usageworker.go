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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	usageCalculateWorker         = appsrv.NewWorkerManager("usageCalculateWorker", 1, 1024, true)
	realTimeUsageCalculateWorker = appsrv.NewWorkerManager("realTimeUsageCalculateWorker", 1, 1024, true)

	usageDirtyMap     = make(map[string]bool, 0)
	usageDirtyMapLock = &sync.Mutex{}
)

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

func (manager *SQuotaBaseManager) PostUsageJob(keys IQuotaKeys, usageChan chan IQuota, realTime bool) {
	if !consts.EnableQuotaCheck() {
		go func() {
			usageChan <- nil
		}()
		return
	}
	key := QuotaKeyString(keys)
	setDirty(key)

	var worker *appsrv.SWorkerManager
	if realTime {
		worker = realTimeUsageCalculateWorker
	} else {
		worker = usageCalculateWorker
	}

	worker.Run(func() {
		ctx := context.Background()

		usage := manager.newQuota()

		if !isDirty(key) {
			if usageChan != nil {
				manager.usageStore.GetQuota(ctx, keys, usage)
				usageChan <- usage
			}
			return
		}

		usage.SetKeys(keys)
		err := usage.FetchUsage(ctx)
		if err != nil {
			log.Debugf("usage.FetchUsage fail %s", err)
			if usageChan != nil {
				usageChan <- nil
			}
			return
		}

		manager.usageStore.SetQuota(ctx, nil, usage)

		clearDirty(key)

		if usageChan != nil {
			usageChan <- usage
		}
	}, nil, nil)
}

func (manager *SQuotaBaseManager) CalculateQuotaUsages(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if !consts.EnableQuotaCheck() {
		return
	}

	log.Infof("CalculateQuotaUsages")
	quota := manager.newQuota()
	keys := quota.GetKeys()
	keyFields := keys.Fields()
	q := manager.Query(keyFields...)

	keyList := make([]IQuotaKeys, 0)
	rows, err := q.Rows()
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("query quotas fail %s", err)
		}
		return
	}
	defer rows.Close()

	for rows.Next() {
		quota := manager.newQuota()
		err = q.Row2Struct(rows, quota)
		if err != nil {
			log.Errorf("Row2Struct fail %s", err)
			return
		}
		keyList = append(keyList, quota.GetKeys())
	}

	var fields []string

	idNameMap, _ := manager.keyList2IdNameMap(ctx, keyList)
	log.Debugf("%s", jsonutils.Marshal(idNameMap))
	for _, keys := range keyList {
		if idNameMap != nil {
			// no error, do check
			if len(fields) == 0 {
				fields = keys.Fields()
			}
			values := keys.Values()
			for i := range fields {
				if strings.HasSuffix(fields[i], "_id") && len(values[i]) > 0 && len(idNameMap[fields[i]][values[i]]) == 0 {
					log.Infof("%s=%s found not exists, delete quota with key %s", fields[i], values[i], jsonutils.Marshal(keys))
					manager.DeleteAllQuotas(ctx, userCred, keys)
					manager.pendingStore.DeleteAllQuotas(ctx, userCred, keys)
					manager.usageStore.DeleteAllQuotas(ctx, userCred, keys)
					continue
				}
			}
		}
		manager.PostUsageJob(keys, nil, false)
	}
}

func (manager *SQuotaBaseManager) keyList2IdNameMap(ctx context.Context, keyList []IQuotaKeys) (map[string]map[string]string, error) {
	idMap := make(map[string]map[string]string)
	var fields []string
	for _, keys := range keyList {
		if len(fields) == 0 {
			fields = keys.Fields()
		}
		values := keys.Values()
		for i := range fields {
			if strings.HasSuffix(fields[i], "_id") && len(values[i]) > 0 {
				if _, ok := idMap[fields[i]]; !ok {
					idMap[fields[i]] = make(map[string]string)
				}
				if _, ok := idMap[fields[i]][values[i]]; !ok {
					idMap[fields[i]][values[i]] = ""
				}
			}
		}
	}
	ret, err := manager.GetIQuotaManager().FetchIdNames(ctx, idMap)
	if err != nil {
		return nil, err
	} else {
		return ret, nil
	}
}

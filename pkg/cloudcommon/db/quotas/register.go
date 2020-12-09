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
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	quotaManagerTable map[reflect.Type]IQuotaManager
)

func init() {
	quotaManagerTable = make(map[reflect.Type]IQuotaManager)

	db.AddUsages = AddUsages
	db.CancelUsages = CancelUsages
	db.CancelPendingUsagesInContext = cancelPendingUsagesInContext
	db.InitPendingUsagesInContext = initPendingUsagesInContext
}

func Register(manager IQuotaManager) {
	obj, _ := db.NewModelObject(manager)
	ele := reflect.Indirect(reflect.ValueOf(obj))
	quotaManagerTable[ele.Type()] = manager
	manager.SetVirtualObject(manager)
}

func getQuotaManager(quota IQuota) IQuotaManager {
	quotaType := reflect.Indirect(reflect.ValueOf(quota)).Type()
	if m, ok := quotaManagerTable[quotaType]; ok {
		return m
	} else {
		log.Fatalf("No manager for quota %s", quotaType.Name())
		return nil
	}
}

func CancelPendingUsage(ctx context.Context, userCred mcclient.TokenCredential, localUsage IQuota, cancelUsage IQuota, save bool) error {
	if !consts.EnableQuotaCheck() {
		return nil
	}

	if localUsage == nil {
		return nil
	}
	manager := getQuotaManager(cancelUsage)
	return manager.cancelPendingUsage(ctx, userCred, localUsage, cancelUsage, save)
}

func CheckSetPendingQuota(ctx context.Context, userCred mcclient.TokenCredential, quota IQuota) error {
	if !consts.EnableQuotaCheck() {
		return nil
	}

	manager := getQuotaManager(quota)
	err := manager.checkSetPendingQuota(ctx, userCred, quota)
	if err != nil {
		return errors.Wrap(err, "manager.checkSetPendingQuota")
	}
	savePendingUsagesInContext(ctx, quota)
	return nil
}

func CancelUsages(ctx context.Context, userCred mcclient.TokenCredential, usages []db.IUsage) {
	if !consts.EnableQuotaCheck() {
		return
	}
	for _, usage := range usages {
		cancelUsage(ctx, userCred, usage.(IQuota))
	}
}

func cancelUsage(ctx context.Context, userCred mcclient.TokenCredential, usage IQuota) {
	manager := getQuotaManager(usage)
	err := manager.cancelUsage(ctx, userCred, usage)
	if err != nil {
		log.Errorf("cancelUsage %s fail: %s", jsonutils.Marshal(usage), err)
	}
}

func AddUsages(ctx context.Context, userCred mcclient.TokenCredential, usages []db.IUsage) {
	if !consts.EnableQuotaCheck() {
		return
	}
	for _, usage := range usages {
		addUsage(ctx, userCred, usage.(IQuota))
	}
}

func addUsage(ctx context.Context, userCred mcclient.TokenCredential, usage IQuota) {
	manager := getQuotaManager(usage)
	err := manager.addUsage(ctx, userCred, usage)
	if err != nil {
		log.Errorf("cancelUsage %s fail: %s", jsonutils.Marshal(usage), err)
	}
}

func GetQuotaCount(ctx context.Context, request IQuota, pendingKeys IQuotaKeys) (int, error) {
	manager := getQuotaManager(request)
	return manager.getQuotaCount(ctx, request, pendingKeys)
}

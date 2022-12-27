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
	"container/list"
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
)

const (
	APP_CONTEXT_KEY_PENDINGUSAGES = appctx.AppContextKey("pendingusages")
)

func initPendingUsagesInContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, APP_CONTEXT_KEY_PENDINGUSAGES, list.New())
}

func appContextPendingUsages(ctx context.Context) []IQuota {
	val := ctx.Value(APP_CONTEXT_KEY_PENDINGUSAGES)
	if val != nil {
		quotaList := val.(*list.List)
		ret := make([]IQuota, 0)
		for e := quotaList.Front(); e != nil; e = e.Next() {
			ret = append(ret, e.Value.(IQuota))
		}
		return ret
	} else {
		return nil
	}
}

func clearPendingUsagesInContext(ctx context.Context) {
	val := ctx.Value(APP_CONTEXT_KEY_PENDINGUSAGES)
	if val != nil {
		quotaList := val.(*list.List)
		for quotaList.Len() > 0 {
			quotaList.Remove(quotaList.Front())
		}
	}
}

func savePendingUsagesInContext(ctx context.Context, quotas ...IQuota) {
	val := ctx.Value(APP_CONTEXT_KEY_PENDINGUSAGES)
	if val != nil {
		quotaList := val.(*list.List)
		for i := range quotas {
			quotaList.PushBack(quotas[i])
		}
	}
}

func cancelPendingUsagesInContext(ctx context.Context, userCred mcclient.TokenCredential) error {
	quotas := appContextPendingUsages(ctx)
	if quotas == nil {
		return nil
	}
	errs := make([]error, 0)
	for i := range quotas {
		// cancel and do not save pending usage
		err := CancelPendingUsage(ctx, userCred, quotas[i], quotas[i], false)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "CancelPendingUsage %s", jsonutils.Marshal(quotas[i])))
		}
	}
	if len(errs) > 0 {
		return errors.NewAggregate(errs)
	}
	clearPendingUsagesInContext(ctx)
	return nil
}

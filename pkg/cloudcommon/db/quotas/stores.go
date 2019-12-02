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

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

const (
	METADATA_KEY = "quota"
)

type SDBQuotaStore struct {
}

func newDBQuotaStore() *SDBQuotaStore {
	return &SDBQuotaStore{}
}

func (store *SDBQuotaStore) GetQuota(ctx context.Context, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, quota IQuota) error {
	var tenant *db.STenant
	var err error

	switch scope {
	case rbacutils.ScopeDomain:
		tenant, err = db.TenantCacheManager.FetchDomainById(ctx, ownerId.GetProjectDomainId())
	default:
		tenant, err = db.TenantCacheManager.FetchTenantById(ctx, ownerId.GetProjectId())
	}

	if err != nil {
		return err
	}
	quotaStr := tenant.GetMetadata(METADATA_KEY, nil)
	quotaJson, _ := jsonutils.ParseString(quotaStr)
	if quotaJson != nil {
		return quotaJson.Unmarshal(quota)
	}
	return nil
}

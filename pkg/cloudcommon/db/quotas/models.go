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
	"reflect"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"

	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SQuotaBaseManager struct {
	db.SResourceBaseManager

	pendingStore IQuotaStore
	usageStore   IQuotaStore

	autoCreate bool
}

const (
	nameSeparator = "."

	quotaKeyword  = "quota"
	quotaKeywords = "quotas"

	quotaUsageKeyword  = "quota-usage"
	quotaUsageKeywords = "quota-usages"
)

func NewQuotaBaseManager(model interface{}, tableName string, pendingStore IQuotaStore, usageStore IQuotaStore) SQuotaBaseManager {
	return SQuotaBaseManager{
		SResourceBaseManager: db.NewResourceBaseManager(model, tableName, quotaKeyword, quotaKeywords),
		pendingStore:         pendingStore,
		usageStore:           usageStore,
		autoCreate:           true,
	}
}

func NewQuotaUsageManager(model interface{}, tableName string) SQuotaBaseManager {
	return SQuotaBaseManager{
		SResourceBaseManager: db.NewResourceBaseManager(model, tableName, quotaUsageKeyword, quotaUsageKeywords),
	}
}

type SQuotaBase struct {
	db.SResourceBase

	DomainId  string `width:"128" charset:"ascii" nullable:"false" primary:"true" list:"user"`
	ProjectId string `name:"tenant_id" width:"128" charset:"ascii" nullable:"false" primary:"true" list:"user"`
	Platform  string `width:"128" charset:"utf8" nullable:"false" primary:"true" list:"user"`
}

func (manager *SQuotaBaseManager) getQuotaInternal(ctx context.Context, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, platform []string, quota IQuota) error {
	q := manager.Query()
	q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	if scope == rbacutils.ScopeProject {
		q = q.Equals("tenant_id", ownerId.GetProjectId())
	} else {
		q = q.IsNullOrEmpty("tenant_id")
	}
	var key string
	if len(platform) > 0 {
		key = strings.Join(platform, nameSeparator)
	}
	q = q.Equals("platform", key)
	err := q.First(quota)
	if err != nil && err != sql.ErrNoRows {
		return err
	} else if err == sql.ErrNoRows && manager.autoCreate {
		quota.FetchSystemQuota(scope, ownerId)
		return manager.setQuotaInternal(ctx, nil, scope, ownerId, platform, quota)
	}
	return nil
}

func (manager *SQuotaBaseManager) setQuotaInternal(ctx context.Context, userCred mcclient.TokenCredential, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, platform []string, quota IQuota) error {
	base := SQuotaBase{}
	base.DomainId = ownerId.GetProjectDomainId()
	if scope == rbacutils.ScopeProject {
		base.ProjectId = ownerId.GetProjectId()
	}
	if len(platform) > 0 {
		base.Platform = strings.Join(platform, nameSeparator)
	}
	base.SetModelManager(manager, quota.(db.IModel))

	if !reflectutils.FillEmbededStructValue(reflect.Indirect(reflect.ValueOf(quota)), reflect.ValueOf(base)) {
		return errors.Error("no embed SBaseQuota")
	}

	return manager.TableSpec().InsertOrUpdate(quota)
}

func (manager *SQuotaBaseManager) deleteQuotaInternal(ctx context.Context, userCred mcclient.TokenCredential, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, platform []string) error {
	q := manager.Query("domain_id", "tenant_id", "platform")
	q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	if scope == rbacutils.ScopeProject {
		q = q.Equals("tenant_id", ownerId.GetProjectId())
	}
	if platform != nil {
		q = q.Equals("platform", strings.Join(platform, nameSeparator))
	}

	rows, err := q.Rows()
	if err != nil {
		if err != sql.ErrNoRows {
			return errors.Wrap(err, "sql.Rows")
		} else {
			return nil
		}
	}

	defer rows.Close()

	for rows.Next() {
		var domainId, tenantId, platform string
		err := rows.Scan(&domainId, &tenantId, &platform)
		if err != nil {
			return errors.Wrap(err, "rows.Scan")
		}
		base := SQuotaBase{
			DomainId:  domainId,
			ProjectId: tenantId,
			Platform:  platform,
		}
		base.SetModelManager(manager, &base)
		err = base.Delete(ctx, nil)
		if err != nil {
			return errors.Wrap(err, "Update")
		}
	}
	return nil
}

func (manager *SQuotaBaseManager) InitializeData() error {
	quotaCnt, err := manager.Query().CountWithError()
	if err != nil {
		return errors.Wrap(err, "SQuotaManager.CountWithError")
	}
	if quotaCnt > 0 {
		// initlaized, quit
		return nil
	}

	metaQuota := newDBQuotaStore()

	tenants := make([]db.STenant, 0)
	err = db.TenantCacheManager.Query().All(&tenants)
	if err != nil && err != sql.ErrNoRows {
		return errors.Wrap(err, "Query")
	}

	for i := range tenants {
		obj := tenants[i]
		var scope rbacutils.TRbacScope
		var ownerId mcclient.IIdentityProvider
		if obj.DomainId == identityapi.KeystoneDomainRoot {
			// domain
			scope = rbacutils.ScopeDomain
			ownerId = &db.SOwnerId{
				DomainId: tenants[i].Id,
				Domain:   tenants[i].Name,
			}
		} else {
			// project
			scope = rbacutils.ScopeProject
			ownerId = &db.SOwnerId{
				DomainId:  tenants[i].DomainId,
				Domain:    tenants[i].Domain,
				ProjectId: tenants[i].Id,
				Project:   tenants[i].Name,
			}
		}

		quota := manager.newQuota()
		err := metaQuota.GetQuota(context.Background(), scope, ownerId, quota)
		if err != nil && err != sql.ErrNoRows {
			log.Errorf("metaQuota.GetQuota error %s for %s", err, ownerId)
			continue
		}
		if quota.IsEmpty() {
			quota.FetchSystemQuota(scope, ownerId)
		}
		baseQuota := SQuotaBase{}
		baseQuota.DomainId = ownerId.GetProjectDomainId()
		baseQuota.ProjectId = ownerId.GetProjectId()
		baseQuota.SetModelManager(manager, quota.(db.IModel))
		reflectutils.FillEmbededStructValue(reflect.Indirect(reflect.ValueOf(quota)), reflect.ValueOf(baseQuota))

		err = manager.TableSpec().Insert(quota)
		if err != nil {
			log.Errorf("insert error %s", err)
			continue
		}
	}

	return nil
}

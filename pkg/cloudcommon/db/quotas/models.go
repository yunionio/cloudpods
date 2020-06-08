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
	"sort"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SQuotaBaseManager struct {
	db.SResourceBaseManager

	pendingStore IQuotaStore
	usageStore   IQuotaStore

	nonNegative bool

	scope rbacutils.TRbacScope
}

func NewQuotaBaseManager(model interface{}, scope rbacutils.TRbacScope, tableName string, pendingStore IQuotaStore, usageStore IQuotaStore, keyword, keywordPlural string) SQuotaBaseManager {
	pendingStore.SetVirtualObject(pendingStore)
	usageStore.SetVirtualObject(usageStore)
	return SQuotaBaseManager{
		SResourceBaseManager: db.NewResourceBaseManager(model, tableName, keyword, keywordPlural),
		pendingStore:         pendingStore,
		usageStore:           usageStore,
		nonNegative:          false,
		scope:                scope,
	}
}

func NewQuotaUsageManager(model interface{}, scope rbacutils.TRbacScope, tableName string, keyword, keywordPlural string) SQuotaBaseManager {
	return SQuotaBaseManager{
		SResourceBaseManager: db.NewResourceBaseManager(model, tableName, keyword, keywordPlural),
		nonNegative:          true,
		scope:                scope,
	}
}

type SQuotaBase struct {
	db.SResourceBase
}

func (manager *SQuotaBaseManager) GetIQuotaManager() IQuotaManager {
	return manager.GetIResourceModelManager().(IQuotaManager)
}

func (manager *SQuotaBaseManager) FetchIdNames(ctx context.Context, idMap map[string]map[string]string) (map[string]map[string]string, error) {
	return idMap, nil
}

func (manager *SQuotaBaseManager) getQuotaByKeys(ctx context.Context, keys IQuotaKeys, quota IQuota) error {
	q := manager.Query()

	fields := keys.Fields()
	values := keys.Values()
	for i := range fields {
		if len(values[i]) == 0 {
			q = q.IsNullOrEmpty(fields[i])
		} else {
			q = q.Equals(fields[i], values[i])
		}
	}

	err := q.First(quota)
	if manager.nonNegative {
		quota.ResetNegative()
	}
	if err != nil {
		return errors.Wrap(err, "q.Query")
	}
	return nil
}

func filterParentByKey(q *sqlchemy.SQuery, fieldName string, value string) *sqlchemy.SQuery {
	if len(value) > 0 {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.IsNullOrEmpty(q.Field(fieldName)),
			sqlchemy.Equals(q.Field(fieldName), value),
		))
	} else {
		q = q.Filter(sqlchemy.IsNullOrEmpty(q.Field(fieldName)))
	}
	return q
}

func filterChildrenByKey(q *sqlchemy.SQuery, fieldName string, value string) *sqlchemy.SQuery {
	if len(value) > 0 {
		q = q.Equals(fieldName, value)
	}
	return q
}

func (manager *SQuotaBaseManager) getQuotasInternal(ctx context.Context, keys IQuotaKeys, isParent bool) ([]IQuota, error) {
	q := manager.Query()

	fields := keys.Fields()
	values := keys.Values()
	for i := range fields {
		if isParent {
			q = filterParentByKey(q, fields[i], values[i])
		} else {
			q = filterChildrenByKey(q, fields[i], values[i])
		}
	}

	rows, err := q.Rows()
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, errors.Wrap(err, "q.Rows")
		}
	}
	defer rows.Close()
	results := make([]IQuota, 0)
	for rows.Next() {
		r := manager.newQuota()
		err := q.Row2Struct(rows, r)
		if err != nil {
			return nil, errors.Wrap(err, "q.Row2Struct")
		}
		if manager.nonNegative {
			r.ResetNegative()
		}
		results = append(results, r)
	}
	sort.Sort(TQuotaList(results))
	return results, nil
}

func (manager *SQuotaBaseManager) setQuotaInternal(ctx context.Context, userCred mcclient.TokenCredential, quota IQuota) error {
	err := manager.TableSpec().InsertOrUpdate(ctx, quota)
	if err != nil {
		return errors.Wrap(err, "InsertOrUpdate")
	}
	if manager.nonNegative {
		quota.ResetNegative()
	}
	return nil
}

func (manager *SQuotaBaseManager) addQuotaInternal(ctx context.Context, userCred mcclient.TokenCredential, diff IQuota) error {
	keys := diff.GetKeys()
	quota := manager.newQuota()
	quota.SetKeys(keys)
	err := manager.getQuotaByKeys(ctx, keys, quota)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			// insert one
		} else {
			return errors.Wrap(err, "manager.getQuotaByKeys")
		}
	}
	quota.Add(diff)
	return manager.setQuotaInternal(ctx, userCred, quota)
}

func (manager *SQuotaBaseManager) subQuotaInternal(ctx context.Context, userCred mcclient.TokenCredential, diff IQuota) error {
	keys := diff.GetKeys()
	quota := manager.newQuota()
	quota.SetKeys(keys)
	err := manager.getQuotaByKeys(ctx, keys, quota)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			// insert one
		} else {
			return errors.Wrap(err, "manager.getQuotaByKeys")
		}
	}
	quota.Sub(diff)
	return manager.setQuotaInternal(ctx, userCred, quota)
}

func (manager *SQuotaBaseManager) deleteQuotaByKeys(ctx context.Context, userCred mcclient.TokenCredential, keys IQuotaKeys) error {
	quota := manager.newQuota()
	quota.SetKeys(keys)
	_, err := db.Update(quota.(db.IModel), func() error {
		return quota.(db.IModel).MarkDelete()
	})
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			return errors.Wrap(err, "Delete")
		}
	}
	return nil
}

func (manager *SQuotaBaseManager) deleteAllQuotas(ctx context.Context, userCred mcclient.TokenCredential, keys IQuotaKeys) error {
	quotas, err := manager.getQuotasInternal(ctx, keys, false)
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			return errors.Wrap(err, "manager.getQuotasInternal")
		} else {
			return nil
		}
	}
	for i := range quotas {
		err := manager.deleteQuotaByKeys(ctx, userCred, quotas[i].GetKeys())
		if err != nil {
			if errors.Cause(err) != sql.ErrNoRows {
				return errors.Wrapf(err, "manager.deleteQuotaByKeys %s", QuotaKeyString(quotas[i].GetKeys()))
			}
		}
	}
	return nil
}

func (manager *SQuotaBaseManager) InitializeData() error {
	q := manager.Query()
	quotaCnt, err := q.CountWithError()
	if err != nil {
		return errors.Wrap(err, "SQuotaManager.CountWithError")
	}
	if quotaCnt > 0 {
		// initlaized, quit
		return nil
	}

	log.Debugf("%s", q.String())

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
		var baseKeys IQuotaKeys
		if manager.scope == rbacutils.ScopeDomain {
			baseKeys = OwnerIdDomainQuotaKeys(ownerId)
		} else {
			baseKeys = OwnerIdProjectQuotaKeys(scope, ownerId)
		}
		if !reflectutils.FillEmbededStructValue(reflect.Indirect(reflect.ValueOf(quota)), reflect.ValueOf(baseKeys)) {
			log.Fatalf("invalid quota??? fail to find SBaseQuotaKey")
		}
		err := metaQuota.GetQuota(context.Background(), scope, ownerId, quota)
		if err != nil && err != sql.ErrNoRows {
			log.Errorf("metaQuota.GetQuota error %s for %s", err, ownerId)
			continue
		}
		if quota.IsEmpty() {
			quota.FetchSystemQuota()
		}
		err = manager.TableSpec().Insert(context.Background(), quota)
		if err != nil {
			log.Errorf("%s insert error %s", manager.KeywordPlural(), err)
			continue
		} else {
			log.Infof("Insert %s", quota)
		}
	}

	return nil
}

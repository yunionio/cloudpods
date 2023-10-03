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

package models

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var TokenCacheManager *STokenCacheManager

func init() {
	TokenCacheManager = &STokenCacheManager{
		SModelBaseManager: db.NewModelBaseManager(
			STokenCache{},
			"token_cache_tbl",
			"token_cache",
			"token_caches",
		),
	}
	TokenCacheManager.SetVirtualObject(TokenCacheManager)
}

type STokenCache struct {
	db.SModelBase

	Token     string    `width:"700" charset:"ascii" nullable:"false" primary:"true"`
	ExpiredAt time.Time `nullable:"false"`
	Valid     bool
	Method    string `width:"32" charset:"ascii"`
	AuditIds  string `width:"700" charset:"utf8" index:"true"`
}

type STokenCacheManager struct {
	db.SModelBaseManager
}

func joinAuditIds(ids []string) string {
	sort.Strings(ids)
	return strings.Join(ids, ",")
}

func (manager *STokenCacheManager) Save(ctx context.Context, token string, expiredAt time.Time, method string, auditIds []string) error {
	return manager.insert(ctx, token, expiredAt, true, method, auditIds)
}

func (manager *STokenCacheManager) Invalidate(ctx context.Context, token string, expiredAt time.Time, method string, auditIds []string) error {
	return manager.insert(ctx, token, expiredAt, false, method, auditIds)
}

func (manager *STokenCacheManager) BatchInvalidate(ctx context.Context, method string, auditIds []string) error {
	invalidQueue := []sCacheCredential{
		{
			Method:   method,
			AuditIds: auditIds,
		},
	}
	for i := 0; i < len(invalidQueue); i++ {
		queues, err := manager.batchInvalidateInternal(ctx, invalidQueue[i])
		if err != nil {
			return errors.Wrap(err, "batchInvalidateInternal")
		}
		if len(queues) > 0 {
			invalidQueue = append(invalidQueue, queues...)
		}
	}
	return nil
}

type sCacheCredential struct {
	Method   string
	AuditIds []string
}

func (manager *STokenCacheManager) batchInvalidateInternal(ctx context.Context, cred sCacheCredential) ([]sCacheCredential, error) {
	q := manager.Query().Equals("method", cred.Method).Equals("audit_ids", joinAuditIds(cred.AuditIds))
	tokens := make([]STokenCache, 0)
	err := db.FetchModelObjects(manager, q, &tokens)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	if len(tokens) == 0 {
		return nil, nil
	}
	queues := make([]sCacheCredential, 0)
	for i := range tokens {
		token := tokens[i]
		queues = append(queues, sCacheCredential{
			Method:   api.AUTH_METHOD_TOKEN,
			AuditIds: []string{token.Token},
		})
	}
	err = manager.TableSpec().GetTableSpec().UpdateBatch(
		map[string]interface{}{
			"valid": false,
		},
		map[string]interface{}{
			"method":    cred.Method,
			"audit_ids": joinAuditIds(cred.AuditIds),
		},
	)
	return queues, errors.Wrap(err, "UpdateBatch")
}

func (manager *STokenCacheManager) insert(ctx context.Context, token string, expiredAt time.Time, valid bool, method string, auditIds []string) error {
	val := STokenCache{
		Token:     token,
		ExpiredAt: expiredAt,
		Valid:     valid,
		Method:    method,
		AuditIds:  joinAuditIds(auditIds),
	}
	err := manager.TableSpec().InsertOrUpdate(ctx, &val)
	return errors.Wrap(err, "InsertOrUpdate")
}

func (manager *STokenCacheManager) IsValid(token string) (bool, error) {
	q := manager.Query().Equals("token", token)
	tokenCache := STokenCache{}
	err := q.First(&tokenCache)
	if err != nil {
		return false, errors.Wrap(err, "Query")
	}
	return tokenCache.Valid, nil
}

func (manager *STokenCacheManager) removeObsolete() error {
	sql := fmt.Sprintf("DELETE FROM `%s` WHERE `expired_at` < ?", manager.TableSpec().Name())
	db := sqlchemy.GetDBWithName(manager.TableSpec().GetDBName())
	now := timeutils.UtcNow()
	_, err := db.Exec(sql, now.Add(-24*time.Hour))
	return errors.Wrap(err, "Exec Delete")
}

func RemoveObsoleteInvalidTokens(ctx context.Context, userCred mcclient.TokenCredential, start bool) {
	err := TokenCacheManager.removeObsolete()
	if err != nil {
		log.Errorf("RemoveObsoleteInvalidTokens fail %s", err)
	}
}

func (manager *STokenCacheManager) FetchInvalidTokens() ([]string, error) {
	q := manager.Query("token").IsFalse("valid")
	tokens := make([]STokenCache, 0)
	err := db.FetchModelObjects(manager, q, &tokens)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	ret := make([]string, len(tokens))
	for i := range tokens {
		ret[i] = tokens[i].Token
	}
	return ret, nil
}

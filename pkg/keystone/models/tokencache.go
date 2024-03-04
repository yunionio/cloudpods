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
	"yunion.io/x/onecloud/pkg/keystone/cache"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

var TokenCacheManager *STokenCacheManager

func init() {
	TokenCacheManager = &STokenCacheManager{
		SStandaloneAnonResourceBaseManager: db.NewStandaloneAnonResourceBaseManager(
			STokenCache{},
			"token_cache_tbl",
			"token_cache",
			"token_caches",
		),
	}
	TokenCacheManager.SetVirtualObject(TokenCacheManager)
	TokenCacheManager.TableSpec().AddIndex(false, "deleted", "valid")
}

type STokenCache struct {
	db.SStandaloneAnonResourceBase

	// Token     string    `width:"700" charset:"ascii" nullable:"false" primary:"true"`
	ExpiredAt time.Time `nullable:"false"`
	Valid     bool

	Method   string `width:"32" charset:"ascii"`
	AuditIds string `width:"256" charset:"utf8" index:"true"`

	UserId    string `width:"128" charset:"ascii" nullable:"false"`
	ProjectId string `width:"128" charset:"ascii" nullable:"true"`
	DomainId  string `width:"128" charset:"ascii" nullable:"true"`

	Source string `width:"16" charset:"ascii"`
	Ip     string `width:"64" charset:"ascii"`
}

type STokenCacheManager struct {
	db.SStandaloneAnonResourceBaseManager
}

func joinAuditIds(ids []string) string {
	sort.Strings(ids)
	return strings.Join(ids, ",")
}

func (manager *STokenCacheManager) Save(ctx context.Context, tokenStr string, expiredAt time.Time, method string, auditIds []string, userId, projId, domainId, source, ip string) error {
	return manager.insert(ctx, tokenStr, expiredAt, true, method, auditIds, userId, projId, domainId, source, ip)
}

func (manager *STokenCacheManager) Invalidate(ctx context.Context, userCred mcclient.TokenCredential, tokenStr string) error {
	token, err := manager.FetchToken(tokenStr)
	if err != nil {
		return errors.Wrap(err, "FetchToken")
	}
	err = token.invalidate(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "token.invalidate")
	}
	return nil
}

func (manager *STokenCacheManager) BatchInvalidateByUserId(ctx context.Context, userCred mcclient.TokenCredential, uid string) error {
	return manager.batchInvalidateInternal(ctx, userCred, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		q = q.Equals("user_id", uid)
		return q
	})
}

func (manager *STokenCacheManager) BatchInvalidate(ctx context.Context, userCred mcclient.TokenCredential, method string, auditIds []string) error {
	return manager.batchInvalidateInternal(ctx, userCred, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		q = q.Equals("method", method).Equals("audit_ids", joinAuditIds(auditIds))
		return q
	})
}

func (manager *STokenCacheManager) batchInvalidateInternal(ctx context.Context, userCred mcclient.TokenCredential, filter func(q *sqlchemy.SQuery) *sqlchemy.SQuery) error {
	q := manager.Query().IsTrue("valid")
	q = filter(q)
	tokens := make([]STokenCache, 0)
	err := db.FetchModelObjects(manager, q, &tokens)
	if err != nil {
		return errors.Wrap(err, "FetchModelObjects")
	}
	if len(tokens) == 0 {
		return nil
	}
	errs := make([]error, 0)
	for i := range tokens {
		token := tokens[i]
		err := token.invalidate(ctx, userCred)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "batchInvalidateInternal token %s", token.Id))
		}
	}
	if len(errs) > 0 {
		return errors.NewAggregate(errs)
	}
	return nil
}

func (manager *STokenCacheManager) insert(ctx context.Context, token string, expiredAt time.Time, valid bool, method string, auditIds []string, userId, projectId, domainId, source, ip string) error {
	val := STokenCache{
		SStandaloneAnonResourceBase: db.SStandaloneAnonResourceBase{
			Id: token,
		},
		ExpiredAt: expiredAt,
		Valid:     valid,
		Method:    method,
		AuditIds:  joinAuditIds(auditIds),
		UserId:    userId,
		ProjectId: projectId,
		DomainId:  domainId,
		Source:    source,
		Ip:        ip,
	}
	val.SetModelManager(manager, &val)
	err := manager.TableSpec().InsertOrUpdate(ctx, &val)
	return errors.Wrap(err, "InsertOrUpdate")
}

func (manager *STokenCacheManager) FetchToken(tokenStr string) (*STokenCache, error) {
	obj, err := manager.FetchById(tokenStr)
	if err != nil {
		return nil, errors.Wrap(err, "FetchById")
	}
	return obj.(*STokenCache), nil
}

func (manager *STokenCacheManager) removeObsolete() error {
	sql := fmt.Sprintf("DELETE FROM %s WHERE expired_at < ?", manager.TableSpec().Name())
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
	q := manager.Query("id").IsFalse("valid")
	tokens := make([]STokenCache, 0)
	err := db.FetchModelObjects(manager, q, &tokens)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	ret := make([]string, len(tokens))
	for i := range tokens {
		ret[i] = tokens[i].Id
	}
	return ret, nil
}

func (token *STokenCache) invalidate(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := TokenCacheManager.BatchInvalidate(ctx, userCred, api.AUTH_METHOD_TOKEN, []string{token.Id})
	if err != nil {
		return errors.Wrapf(err, "BatchInvalidate subtoken %s", token.Id)
	}

	_, err = db.Update(token, func() error {
		token.Valid = false
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "update")
	}

	cache.Remove(token.Id)

	logclient.AddActionLogWithContext(ctx, token, logclient.ACT_DELETE, token.GetShortDesc(ctx), userCred, true)

	return nil
}

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

package db

import (
	"context"
	"database/sql"
	"fmt"
	"runtime/debug"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	DefaultUserFetcher func(ctx context.Context, id string) (*SUser, error)
)

type SUserCacheManager struct {
	SKeystoneCacheObjectManager
}

type SUser struct {
	SKeystoneCacheObject
}

func (user *SUser) GetModelManager() IModelManager {
	return UserCacheManager
}

var UserCacheManager *SUserCacheManager

func init() {
	UserCacheManager = &SUserCacheManager{
		NewKeystoneCacheObjectManager(SUser{}, "users_cache_tbl", "user", "users")}
	// log.Debugf("initialize user cache manager %s", UserCacheManager.KeywordPlural())
	UserCacheManager.SetVirtualObject(UserCacheManager)

	DefaultUserFetcher = UserCacheManager.FetchUserByIdOrName
}

func (manager *SUserCacheManager) updateUserCache(userCred mcclient.TokenCredential) {
	manager.Save(context.Background(), userCred.GetUserId(), userCred.GetUserName(),
		userCred.GetDomainId(), userCred.GetDomainName())
}

func (manager *SUserCacheManager) FetchUserByIdOrName(ctx context.Context, idStr string) (*SUser, error) {
	return manager.fetchUser(ctx, idStr, false, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		if stringutils2.IsUtf8(idStr) {
			return q.Equals("name", idStr)
		} else {
			return q.Filter(sqlchemy.OR(
				sqlchemy.Equals(q.Field("id"), idStr),
				sqlchemy.Equals(q.Field("name"), idStr),
			))
		}
	})
}

func (manager *SUserCacheManager) FetchUserById(ctx context.Context, idStr string) (*SUser, error) {
	return manager.fetchUser(ctx, idStr, false, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Filter(sqlchemy.Equals(q.Field("id"), idStr))
	})
}

func (manager *SUserCacheManager) FetchUserByName(ctx context.Context, idStr string) (*SUser, error) {
	return manager.fetchUser(ctx, idStr, false, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Filter(sqlchemy.Equals(q.Field("name"), idStr))
	})
}

func (manager *SUserCacheManager) fetchUser(
	ctx context.Context, idStr string, noExpireCheck bool,
	filter func(*sqlchemy.SQuery) *sqlchemy.SQuery,
) (*SUser, error) {
	q := manager.Query()
	q = filter(q)
	tobj, err := NewModelObject(manager)
	if err != nil {
		return nil, errors.Wrap(err, "NewModelObject")
	}
	err = q.First(tobj)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "query")
	} else if tobj != nil {
		user := tobj.(*SUser)
		if noExpireCheck || !user.IsExpired() {
			return user, nil
		}
	}
	return manager.FetchUserFromKeystone(ctx, idStr)
}

func (manager *SUserCacheManager) FetchUserFromKeystone(ctx context.Context, idStr string) (*SUser, error) {
	if len(idStr) == 0 {
		log.Debugf("fetch empty user!!!!\n%s", debug.Stack())
		return nil, fmt.Errorf("Empty idStr")
	}

	// It's to query the full list of users(contains other domain's ones and system ones)
	query := jsonutils.NewDict()
	query.Set("scope", jsonutils.NewString("system"))
	query.Set("system", jsonutils.JSONTrue)

	s := auth.GetAdminSession(ctx, consts.GetRegion(), "v1")
	user, err := modules.UsersV3.GetById(s, idStr, query)
	if err != nil {
		if je, ok := err.(*httputils.JSONClientError); ok && je.Code == 404 {
			user, err = modules.UsersV3.GetByName(s, idStr, query)
			if je, ok := err.(*httputils.JSONClientError); ok && je.Code == 404 {
				return nil, sql.ErrNoRows
			}
		}
		if err != nil {
			log.Errorf("fetch user %s fail %s", idStr, err)
			return nil, errors.Wrap(err, "modules.UsersV3.Get")
		}
	}
	id, _ := user.GetString("id")
	name, _ := user.GetString("name")
	domainId, _ := user.GetString("domain_id")
	domainNmae, _ := user.GetString("project_domain")
	return manager.Save(ctx, id, name, domainId, domainNmae)
}

func (manager *SUserCacheManager) Save(ctx context.Context, idStr string, name string, domainId string, domain string) (*SUser, error) {
	lockman.LockRawObject(ctx, manager.KeywordPlural(), idStr)
	defer lockman.ReleaseRawObject(ctx, manager.KeywordPlural(), idStr)

	objo, err := manager.FetchById(idStr)
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("FetchUserbyId fail %s", err)
		return nil, err
	}
	if err == nil {
		obj := objo.(*SUser)
		_, err = Update(obj, func() error {
			obj.Id = idStr
			obj.Name = name
			obj.Domain = domain
			obj.DomainId = domainId
			obj.LastCheck = time.Now().UTC()
			return nil
		})
		if err != nil {
			return nil, err
		} else {
			return obj, nil
		}
	} else {
		objm, err := NewModelObject(manager)
		obj := objm.(*SUser)
		obj.Id = idStr
		obj.Name = name
		obj.Domain = domain
		obj.DomainId = domainId
		obj.LastCheck = time.Now().UTC()
		err = manager.TableSpec().InsertOrUpdate(ctx, obj)
		if err != nil {
			return nil, err
		} else {
			return obj, nil
		}
	}
}

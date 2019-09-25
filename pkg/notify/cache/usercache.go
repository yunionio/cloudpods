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

package cache

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type SUserCacheManager struct {
	db.SKeystoneCacheObjectManager
}

type SUser struct {
	db.SKeystoneCacheObject
}

func (user *SUser) GetModelManager() db.IModelManager {
	return UserCacheManager
}

var UserCacheManager *SUserCacheManager

func init() {
	UserCacheManager = &SUserCacheManager{
		db.NewKeystoneCacheObjectManager(SUser{}, "users_cache_tbl", "user", "users")}
	// log.Debugf("initialize user cache manager %s", UserCacheManager.KeywordPlural())
	UserCacheManager.SetVirtualObject(UserCacheManager)
}

func RegistUserCredCacheUpdater() {
	auth.RegisterAuthHook(onAuthCompleteUpdateCache)
}

func onAuthCompleteUpdateCache(userCred mcclient.TokenCredential) {
	UserCacheManager.updateUserCache(userCred)
}

func (ucm *SUserCacheManager) updateUserCache(userCred mcclient.TokenCredential) {
	ucm.Save(context.Background(), userCred.GetUserId(), userCred.GetUserName(),
		userCred.GetDomainId())
}

func (ucm *SUserCacheManager) FetchUserByIdOrName(idStr string) (*SUser, error) {
	obj, err := ucm.SKeystoneCacheObjectManager.FetchByIdOrName(nil, idStr)
	if err != nil {
		return nil, err
	}
	return obj.(*SUser), nil
}

func (ucm *SUserCacheManager) FetchUserById(idStr string) (*SUser, error) {
	obj, err := ucm.SKeystoneCacheObjectManager.FetchById(idStr)
	if err != nil {
		return nil, err
	}
	return obj.(*SUser), nil
}

func (ucm *SUserCacheManager) FetchUserByName(idStr string) (*SUser, error) {
	obj, err := ucm.SKeystoneCacheObjectManager.FetchByName(nil, idStr)
	if err != nil {
		return nil, err
	}
	return obj.(*SUser), nil
}

func (ucm *SUserCacheManager) Save(ctx context.Context, idStr string, name string, domainId string) (*SUser, error) {
	lockman.LockRawObject(ctx, ucm.KeywordPlural(), idStr)
	defer lockman.ReleaseRawObject(ctx, ucm.KeywordPlural(), idStr)

	objo, err := ucm.FetchById(idStr)
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("FetchTenantbyId fail %s", err)
		return nil, err
	}
	now := time.Now().UTC()
	if err == nil {
		obj := objo.(*SUser)
		if obj.Id == idStr && obj.Name == name && obj.DomainId == domainId {
			db.Update(obj, func() error {
				obj.LastCheck = now
				return nil
			})
			return obj, nil
		}
		_, err = db.Update(obj, func() error {
			obj.Id = idStr
			obj.Name = name
			obj.DomainId = domainId
			obj.LastCheck = now
			return nil
		})
		if err != nil {
			return nil, err
		} else {
			return obj, nil
		}
	} else {
		objm, err := db.NewModelObject(ucm)
		obj := objm.(*SUser)
		obj.Id = idStr
		obj.Name = name
		obj.DomainId = domainId
		obj.LastCheck = now
		err = ucm.TableSpec().Insert(obj)
		if err != nil {
			return nil, err
		} else {
			return obj, nil
		}
	}
}

func (ucm *SUserCacheManager) fetchUserFromKeystone(ctx context.Context, idStr string) (*SUser, error) {
	if len(idStr) == 0 {
		return nil, fmt.Errorf("Empty idStr")
	}
	s := auth.GetAdminSession(ctx, consts.GetRegion(), "v3")
	user, err := modules.UsersV3.GetById(s, idStr, nil)
	if err != nil {
		if je, ok := err.(*httputils.JSONClientError); ok && je.Code == 404 {
			return nil, sql.ErrNoRows
		}
		log.Errorf("fetch project %s fail %s", idStr, err)
		return nil, errors.Wrap(err, "modules.Projects.Get")
	}
	userId, _ := user.GetString("id")
	userName, _ := user.GetString("name")
	domainId, _ := user.GetString("domain_id")
	return ucm.Save(ctx, userId, userName, domainId)
}

func (ucm *SUserCacheManager) FetchUsersByIDs(ctx context.Context, ids []string) (map[string]SUser, error) {
	q := ucm.Query().In("id", ids)
	users := make([]SUser, 0)
	err := db.FetchModelObjects(ucm, q, &users)
	if err != nil {
		return nil, err
	}
	ret := make(map[string]SUser)

	for i := range users {
		ret[users[i].Id] = users[i]
	}

	// check that id is exist
	for _, id := range ids {
		if _, ok := ret[id]; ok {
			continue
		}
		user, err := ucm.fetchUserFromKeystone(ctx, id)
		if err != nil {
			continue
		}
		ret[id] = *user
	}
	return ret, nil
}

func (ucm *SUserCacheManager) FetchUserByID(ctx context.Context, idStr string, noExpireCheck bool) (*SUser, error) {

	q := ucm.Query().Equals("id", idStr)
	uobj, err := db.NewModelObject(ucm)
	if err != nil {
		return nil, errors.Wrap(err, "NewModelObject")
	}
	err = q.First(uobj)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "query")
	} else if uobj != nil {
		user := uobj.(*SUser)
		if noExpireCheck || !user.IsExpired() {
			return user, nil
		}
	}
	return ucm.fetchUserFromKeystone(ctx, idStr)
}

func (ucm *SUserCacheManager) FetchUserLikeName(ctx context.Context, name string, noExpireCheck bool) ([]SUser,
	error) {

	if !noExpireCheck {
		// todo
		return nil, fmt.Errorf("FetchUserLikeName with check Not Implement")
	}
	q := ucm.Query().Like("name", "%"+name+"%")
	users := make([]SUser, 0)
	err := db.FetchModelObjects(ucm, q, &users)
	if err != nil {
		return nil, err
	}
	return users, nil
}

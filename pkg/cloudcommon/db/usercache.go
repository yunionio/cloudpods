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

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
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
	UserCacheManager = &SUserCacheManager{NewKeystoneCacheObjectManager(SUser{}, "users_cache_tbl", "user", "users")}
	// log.Debugf("initialize user cache manager %s", UserCacheManager.KeywordPlural())
}

func (manager *SUserCacheManager) FetchUserByIdOrName(idStr string) (*SUser, error) {
	obj, err := manager.SKeystoneCacheObjectManager.FetchByIdOrName(nil, idStr)
	if err != nil {
		return nil, err
	}
	return obj.(*SUser), nil
}

func (manager *SUserCacheManager) FetchUserById(idStr string) (*SUser, error) {
	obj, err := manager.SKeystoneCacheObjectManager.FetchById(idStr)
	if err != nil {
		return nil, err
	}
	return obj.(*SUser), nil
}

func (manager *SUserCacheManager) FetchUserByName(idStr string) (*SUser, error) {
	obj, err := manager.SKeystoneCacheObjectManager.FetchByName(nil, idStr)
	if err != nil {
		return nil, err
	}
	return obj.(*SUser), nil
}

func (manager *SUserCacheManager) Save(ctx context.Context, idStr string, name string, domainId string, domain string) (*SUser, error) {
	lockman.LockRawObject(ctx, manager.KeywordPlural(), idStr)
	defer lockman.ReleaseRawObject(ctx, manager.KeywordPlural(), idStr)

	obj, err := manager.FetchUserById(idStr)
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("FetchTenantbyId fail %s", err)
		return nil, err
	}
	if err == nil {
		_, err = Update(obj, func() error {
			obj.Id = idStr
			obj.Name = name
			obj.Domain = domain
			obj.DomainId = domainId
			return nil
		})
		if err != nil {
			return nil, err
		} else {
			return obj, nil
		}
	} else {
		objm, err := NewModelObject(manager)
		obj = objm.(*SUser)
		obj.Id = idStr
		obj.Name = name
		obj.Domain = domain
		obj.DomainId = domainId
		err = manager.TableSpec().Insert(obj)
		if err != nil {
			return nil, err
		} else {
			return obj, nil
		}
	}
}

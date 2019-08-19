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
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type SUserGroupCacheManager struct {
	db.SResourceBaseManager
}

type SUserGroup struct {
	db.SResourceBase
	UserId    string
	GroupId   string
	LastCheck time.Time `nullable:"false"`
}

func (ug *SUserGroup) GetModelManager() db.IModelManager {
	return UserGroupCacheManager
}

var UserGroupCacheManager *SUserGroupCacheManager

func init() {
	UserGroupCacheManager = &SUserGroupCacheManager{db.NewResourceBaseManager(
		SUserGroup{},
		"user_group_cache_tbl",
		"usergroup",
		"usergroups",
	)}
}

func (ug *SUserGroup) IsExpired() bool {
	if ug.LastCheck.IsZero() {
		return true
	}
	now := time.Now().UTC()
	if ug.LastCheck.Add(consts.GetTenantCacheExpireSeconds()).Before(now) {
		return true
	}
	return false
}

func (manager *SUserGroupCacheManager) FetchByGroupId(ctx context.Context, groupId string) ([]SUserGroup, error) {
	q := manager.Query().Equals("gourp_id", groupId)
	ugs := make([]SUserGroup, 0)
	err := db.FetchModelObjects(manager, q, &ugs)
	if err != nil {
		return nil, err
	}
	var needSync bool
	if len(ugs) == 0 {
		needSync = true
	}
	now := time.Now().UTC()
	expireTime := now.Add(-consts.GetTenantCacheExpireSeconds())
	for i := range ugs {
		if ugs[i].LastCheck.Before(expireTime) {
			needSync = true
			break
		}
	}
	if !needSync {
		return ugs, nil
	}
	ugs, syncResult, err := manager.Sync(ctx, ugs, groupId)
	if err != nil {
		return nil, err
	}
	if syncResult.IsError() {
		log.Errorf(syncResult.Result())
	}
	return ugs, nil
}

func (manager *SUserGroupCacheManager) Sync(ctx context.Context, ugCache []SUserGroup, groupId string) ([]SUserGroup,
	compare.SyncResult, error) {
	lockman.LockRawObject(ctx, manager.KeywordPlural(), groupId)
	defer lockman.ReleaseRawObject(ctx, manager.KeywordPlural(), groupId)

	s := auth.GetAdminSession(ctx, consts.GetRegion(), "v3")
	syncResult := compare.SyncResult{}
	users, err := modules.Groups.GetUsers(s, groupId)
	if err != nil {
		return nil, syncResult, errors.Wrap(err, "fetch users by group id from keystone failed")
	}
	newUgCache := make([]SUserGroup, len(users.Data))
	for i := range users.Data {
		userId, _ := users.Data[i].GetString("id")
		newUgCache[i] = SUserGroup{
			UserId:  userId,
			GroupId: groupId,
		}
	}
	added := make([]SUserGroup, 0)
	removed := make([]SUserGroup, 0)
	commondb := make([]SUserGroup, 0)
	compareSets(ugCache, newUgCache, &added, &removed, &commondb)
	now := time.Now().UTC()
	for i := range added {
		added[i].LastCheck = now
		err := manager.TableSpec().Insert(&added[i])
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncResult.Add()
		}
	}

	for i := range commondb {
		ug := &commondb[i]
		_, err := db.Update(ug, func() error {
			ug.LastCheck = now
			return nil
		})
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncResult.Update()
		}
	}

	for i := range removed {
		ug := &removed[i]
		_, err := db.Update(ug, func() error {
			return ug.MarkDelete()
		})
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	return newUgCache, syncResult, nil
}

func compareSets(dbs, remotes []SUserGroup, added, removed, commondb *[]SUserGroup) {
	dbmap := make(map[string]SUserGroup)
	for i := range dbs {
		dbmap[dbs[i].UserId] = dbs[i]
	}

	for i := range remotes {
		userId := remotes[i].UserId
		if _, ok := dbmap[userId]; ok {
			*commondb = append(*commondb, remotes[i])
		} else {
			*added = append(*added, remotes[i])
		}
		delete(dbmap, userId)
	}
	for _, v := range dbmap {
		*removed = append(*removed, v)
	}
}

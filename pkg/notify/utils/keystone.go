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

package utils

import (
	"context"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/notify/cache"
)

func GetUserByIDOrName(ctx context.Context, idStr string) (*cache.SUser, error) {
	return cache.UserCacheManager.FetchUserByIDOrName(ctx, idStr)
}

func GetUsersWithoutRemote(idStr []string) ([]cache.SUser, error) {
	q := cache.UserCacheManager.Query()
	q = q.Filter(sqlchemy.OR(sqlchemy.In(q.Field("id"), idStr), sqlchemy.In(q.Field("name"), idStr)))
	users := make([]cache.SUser, 0, 1)
	err := db.FetchModelObjects(cache.UserCacheManager, q, &users)
	if err != nil {
		return nil, errors.Wrapf(err, "fetch user cache failed")
	}
	return users, nil
}

func GetUserIdsLikeName(ctx context.Context, name string) ([]string, error) {
	users, err := cache.UserCacheManager.FetchUserLikeName(ctx, name, true)
	if err != nil {
		return nil, err
	}
	ret := make([]string, len(users))
	for i := range users {
		ret[i] = users[i].Id
	}
	return ret, nil
}

func GetUsersByGroupID(ctx context.Context, gid string) ([]string, error) {
	ret, err := cache.UserGroupCacheManager.FetchByGroupId(ctx, gid)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(ret))
	for i := range ret {
		ids[i] = ret[i].UserId
	}
	return ids, nil
}

func GetUsernameByID(ctx context.Context, id string) (string, error) {
	user, err := GetUserByIDOrName(ctx, id)
	if err != nil {
		return "", err
	}
	return user.Name, nil
}

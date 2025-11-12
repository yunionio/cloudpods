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
	"database/sql"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// +onecloud:swagger-gen-ignore
type SUserLoginManager struct {
	db.SModelBaseManager
}

var UserLoginManager *SUserLoginManager

func init() {
	UserLoginManager = &SUserLoginManager{
		SModelBaseManager: db.NewModelBaseManager(
			SUserLogin{},
			"user_login",
			"user_login",
			"user_logins",
		),
	}
	UserLoginManager.SetVirtualObject(UserLoginManager)
}

// +onecloud:swagger-gen-ignore
type SUserLogin struct {
	db.SModelBase

	UserId string `width:"64" charset:"ascii" nullable:"false" primary:"true"`

	// 上次登录时间
	LastActiveAt time.Time `nullable:"true" list:"domain"`
	// 上次用户登录IP
	LastLoginIp string `nullable:"true" list:"domain"`
	// 上次用户登录方式，可能值有：web（web控制台），cli（命令行climc），API（api）
	LastLoginSource string `nullable:"true" list:"domain"`
}

func (manager *SUserLoginManager) fetchUserLogin(userId string) (*SUserLogin, error) {
	userLogin := &SUserLogin{}
	userLogin.SetModelManager(manager, userLogin)
	err := manager.Query().Equals("user_id", userId).First(userLogin)
	if err != nil {
		return nil, errors.Wrap(err, "Query")
	}
	return userLogin, nil
}

func (manager *SUserLoginManager) traceLoginEvent(ctx context.Context, userId string, authCtx mcclient.SAuthContext) error {
	userLogin, err := manager.fetchUserLogin(userId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			// do insert
			userLogin := &SUserLogin{
				UserId:          userId,
				LastActiveAt:    time.Now().UTC(),
				LastLoginIp:     authCtx.Ip,
				LastLoginSource: authCtx.Source,
			}
			err := manager.TableSpec().Insert(ctx, userLogin)
			if err != nil {
				return errors.Wrap(err, "Insert")
			}
			return nil
		} else {
			return errors.Wrap(err, "fetchUserLogin")
		}
	}
	// only save web console login record
	if userLogin.LastActiveAt.IsZero() || utils.IsInArray(authCtx.Source, []string{mcclient.AuthSourceWeb}) {
		_, err := db.Update(userLogin, func() error {
			userLogin.LastActiveAt = time.Now().UTC()
			userLogin.LastLoginIp = authCtx.Ip
			userLogin.LastLoginSource = authCtx.Source
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "Update")
		}
	}
	return nil
}

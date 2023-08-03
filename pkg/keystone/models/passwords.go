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
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	notifyapi "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/options"
	o "yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/notify"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SPasswordManager struct {
	db.SResourceBaseManager
}

var PasswordManager *SPasswordManager

func init() {
	PasswordManager = &SPasswordManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SPassword{},
			"password",
			"password",
			"passwords",
		),
	}
	PasswordManager.SetVirtualObject(PasswordManager)
}

/*
+----------------+--------------+------+-----+---------+----------------+
| Field          | Type         | Null | Key | Default | Extra          |
+----------------+--------------+------+-----+---------+----------------+
| id             | int(11)      | NO   | PRI | NULL    | auto_increment |
| local_user_id  | int(11)      | NO   | MUL | NULL    |                |
| password       | varchar(128) | YES  |     | NULL    |                |
| expires_at     | datetime     | YES  |     | NULL    |                |
| self_service   | tinyint(1)   | NO   |     | 0       |                |
| password_hash  | varchar(255) | YES  |     | NULL    |                |
| created_at_int | bigint(20)   | NO   |     | 0       |                |
| expires_at_int | bigint(20)   | YES  |     | NULL    |                |
| created_at     | datetime     | NO   |     | NULL    |                |
+----------------+--------------+------+-----+---------+----------------+
*/

type SPassword struct {
	db.SResourceBase

	Id           int       `primary:"true" auto_increment:"true"`
	LocalUserId  int       `nullable:"false" index:"true"`
	Password     string    `width:"128" charset:"ascii" nullable:"true"`
	ExpiresAt    time.Time `nullable:"true"`
	SelfService  bool      `nullable:"false" default:"false"`
	PasswordHash string    `width:"255" charset:"ascii" nullable:"true"`
	CreatedAtInt int64     `nullable:"false" default:"0"`
	ExpiresAtInt int64     `nullable:"true"`
}

func shaPassword(passwd string) string {
	shaOut := sha256.Sum224([]byte(passwd))
	return hex.EncodeToString(shaOut[:])
}

func (manager *SPasswordManager) CreateByInsertOrUpdate() bool {
	return false
}

func (manager *SPasswordManager) FetchLastPassword(localUserId int) (*SPassword, error) {
	passes, err := manager.fetchByLocaluserId(localUserId)
	if err != nil {
		return nil, err
	}
	if len(passes) == 0 {
		return nil, nil
	}
	return &passes[0], nil
}

func (manager *SPasswordManager) fetchByLocaluserId(localUserId int) ([]SPassword, error) {
	passes := make([]SPassword, 0)
	passwords := manager.Query().SubQuery()

	q := passwords.Query().Equals("local_user_id", localUserId)
	q = q.Desc(passwords.Field("created_at_int"))
	err := db.FetchModelObjects(manager, q, &passes)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}

	return passes, nil
}

func validatePasswordComplexity(password string) error {
	if options.Options.PasswordMinimalLength > 0 && len(password) < o.Options.PasswordMinimalLength {
		return errors.Wrap(httperrors.ErrWeakPassword, "too simple password")
	}
	if options.Options.PasswordCharComplexity > 0 {
		complexity := options.Options.PasswordCharComplexity
		if complexity > 4 {
			complexity = 4
		}
		if stringutils2.GetCharTypeCount(password) < complexity {
			return errors.Wrap(httperrors.ErrWeakPassword, "too simple password")
		}
	}
	return nil
}

func (manager *SPasswordManager) validatePassword(localUserId int, password string, skipHistoryCheck bool) error {
	err := validatePasswordComplexity(password)
	if err != nil {
		return errors.Wrap(err, "validatePasswordComplexity")
	}
	if !skipHistoryCheck && options.Options.PasswordUniqueHistoryCheck > 0 {
		shaPass := shaPassword(password)
		histPasses, err := manager.fetchByLocaluserId(localUserId)
		if err != nil {
			return errors.Wrap(err, "manager.fetchByLocaluserId")
		}
		for i := 0; i < len(histPasses) && i < options.Options.PasswordUniqueHistoryCheck; i += 1 {
			if histPasses[i].Password == shaPass {
				return errors.Error("repeated password")
			}
		}
	}
	return nil
}

func (manager *SPasswordManager) savePassword(localUserId int, password string, isSystemAccount bool) error {
	hash, err := seclib2.BcryptPassword(password)
	if err != nil {
		return errors.Wrap(err, "seclib2.BcryptPassword")
	}
	rec := &SPassword{
		LocalUserId:  localUserId,
		PasswordHash: hash,
		Password:     shaPassword(password),
	}
	rec.SetModelManager(PasswordManager, rec)
	now := time.Now()
	rec.CreatedAtInt = now.UnixNano() / 1000
	if options.Options.PasswordExpirationSeconds > 0 && !isSystemAccount {
		rec.ExpiresAt = now.Add(time.Second * time.Duration(options.Options.PasswordExpirationSeconds))
		rec.ExpiresAtInt = rec.ExpiresAt.UnixNano() / 1000
	}
	err = manager.TableSpec().Insert(context.TODO(), rec)
	if err != nil {
		return errors.Wrap(err, "Insert")
	}
	return nil
}

func (manager *SPasswordManager) delete(localUserId int) error {
	recs, err := manager.fetchByLocaluserId(localUserId)
	if err != nil {
		return errors.Wrap(err, "manager.fetchByLocaluserId")
	}
	for i := range recs {
		_, err = db.Update(&recs[i], func() error {
			return recs[i].MarkDelete()
		})
		if err != nil {
			return errors.Wrap(err, "recs[i].MarkDelete")
		}
	}
	return nil
}

func (passwd *SPassword) IsExpired() bool {
	if !passwd.ExpiresAt.IsZero() && passwd.ExpiresAt.Before(time.Now()) {
		return true
	}
	return false
}

// 定时任务判断用户是否需要密码过期通知
func CheckAllUserPasswordIsExpired(ctx context.Context, userCred mcclient.TokenCredential, startRun bool) {
	pwds := []SPassword{}
	pwdQ := PasswordManager.Query()
	pwdQ = pwdQ.Desc("created_at")
	err := db.FetchModelObjects(PasswordManager, pwdQ, &pwds)
	if err != nil {
		log.Errorln("fetch Password error:", err)
		return
	}
	hasCheckedPwd := make(map[int]struct{})
	for _, pwd := range pwds {
		if _, isExist := hasCheckedPwd[pwd.LocalUserId]; isExist {
			continue
		}
		if pwd.ExpiresAt.IsZero() {
			continue
		}
		hasCheckedPwd[pwd.LocalUserId] = struct{}{}
		err = pwd.NeedSendNotify(ctx, userCred)
		if err != nil {
			log.Errorln(errors.Wrap(err, "pwd.NeedSendNotify"))
		}
	}
}

func (pwd *SPassword) NeedSendNotify(ctx context.Context, userCred mcclient.TokenCredential) error {
	expireTime := time.Date(pwd.ExpiresAt.Year(), pwd.ExpiresAt.Month(), pwd.ExpiresAt.Day(), 0, 0, 0, 0, time.Local)
	nowTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Local)

	sub := expireTime.Sub(nowTime)
	subDay := int(sub.Hours() / 24)
	s := GetDefaultClientSession(ctx, userCred, options.Options.Region)
	resp, err := notify.NotifyTopic.List(s, jsonutils.Marshal(map[string]interface{}{
		"filter": fmt.Sprintf("name.equals('%s')", notifyapi.DefaultPasswordExpire),
		"scope":  "system",
	}))
	if err != nil {
		return errors.Wrap(err, "list topics")
	}
	topics := []notifyapi.TopicDetails{}
	err = jsonutils.Update(&topics, resp.Data)
	if err != nil {
		return errors.Wrap(err, "update topic")
	}
	if len(topics) != 1 {
		return errors.Wrapf(errors.ErrNotSupported, "len topics :%d", len(topics))
	}

	if utils.IsInArray(subDay, topics[0].AdvanceDays) {
		localUser, err := LocalUserManager.fetchLocalUser("", "", pwd.LocalUserId)
		if err != nil {
			return errors.Wrap(err, "fetchLocalUser error:")
		}
		pwd.EventNotify(ctx, userCred, notifyapi.ActionPasswordExpireSoon, localUser.Name, subDay)
	}
	return nil
}

// 密码即将失效消息通知
func (pwd *SPassword) EventNotify(ctx context.Context, userCred mcclient.TokenCredential, action notifyapi.SAction, userName string, advanceDays int) {
	resourceType := notifyapi.TOPIC_RESOURCE_USER

	detailsDecro := func(ctx context.Context, details *jsonutils.JSONDict) {
		details.Set("account", jsonutils.NewString(userName))
		details.Set("advance_days", jsonutils.NewInt(int64(advanceDays)))
	}
	pwd.Password = ""

	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:                 pwd,
		ObjDetailsDecorator: detailsDecro,
		ResourceType:        resourceType,
		Action:              action,
		AdvanceDays:         advanceDays,
	})
}

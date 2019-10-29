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
	"database/sql"
	"fmt"
	"time"

	"github.com/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

type SLocalUserManager struct {
	db.SResourceBaseManager
}

var LocalUserManager *SLocalUserManager

func init() {
	LocalUserManager = &SLocalUserManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SLocalUser{},
			"local_user",
			"local_user",
			"local_users",
		),
	}
	LocalUserManager.SetVirtualObject(LocalUserManager)
}

/*
+-------------------+--------------+------+-----+---------+----------------+
| Field             | Type         | Null | Key | Default | Extra          |
+-------------------+--------------+------+-----+---------+----------------+
| id                | int(11)      | NO   | PRI | NULL    | auto_increment |
| user_id           | varchar(64)  | NO   | UNI | NULL    |                |
| domain_id         | varchar(64)  | NO   | MUL | NULL    |                |
| name              | varchar(255) | NO   |     | NULL    |                |
| failed_auth_count | int(11)      | YES  |     | NULL    |                |
| failed_auth_at    | datetime     | YES  |     | NULL    |                |
+-------------------+--------------+------+-----+---------+----------------+
*/

type SLocalUser struct {
	db.SResourceBase

	Id              int       `nullable:"false" primary:"true" auto_increment:"true"`
	UserId          string    `width:"64" charset:"ascii" nullable:"false" index:"true"`
	DomainId        string    `width:"64" charset:"ascii" nullable:"false" index:"true"`
	Name            string    `width:"255" charset:"utf8" nullable:"false"`
	FailedAuthCount int       `nullable:"true"`
	FailedAuthAt    time.Time `nullable:"true"`
}

func (user *SLocalUser) GetId() string {
	return fmt.Sprintf("%d", user.Id)
}

func (user *SLocalUser) GetName() string {
	return user.Name
}

func (manager *SLocalUserManager) FetchLocalUser(usrExt *api.SUserExtended) (*SLocalUser, error) {
	localUser := SLocalUser{}
	localUser.SetModelManager(manager, &localUser)
	q := manager.Query().Equals("id", usrExt.LocalId)
	err := q.First(&localUser)
	if err != nil {
		return nil, errors.Wrap(err, "Query")
	}
	return &localUser, nil
}

func (manager *SLocalUserManager) register(userId string, domainId string, name string) (*SLocalUser, error) {
	localUser := SLocalUser{}
	localUser.SetModelManager(manager, &localUser)

	q := manager.Query().Equals("user_id", userId).Equals("domain_id", domainId)
	err := q.First(&localUser)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "Query")
	}
	if err == nil {
		return &localUser, nil
	}
	localUser.UserId = userId
	localUser.DomainId = domainId
	localUser.Name = name

	err = manager.TableSpec().Insert(&localUser)
	if err != nil {
		return nil, errors.Wrap(err, "Insert")
	}

	return &localUser, nil
}

func (manager *SLocalUserManager) delete(userId string, domainId string) (*SLocalUser, error) {
	localUser := SLocalUser{}
	localUser.SetModelManager(manager, &localUser)

	q := manager.Query().Equals("user_id", userId).Equals("domain_id", domainId)
	err := q.First(&localUser)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "Query")
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}

	_, err = db.Update(&localUser, func() error {
		return localUser.MarkDelete()
	})

	if err != nil {
		return nil, errors.Wrap(err, "MarkDelete")
	}

	return &localUser, nil
}

func (usr *SLocalUser) SaveFailedAuth() error {
	_, err := db.Update(usr, func() error {
		usr.FailedAuthCount += 1
		usr.FailedAuthAt = time.Now()
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "Update")
	}
	return nil
}

func (usr *SLocalUser) ClearFailedAuth() error {
	_, err := db.Update(usr, func() error {
		usr.FailedAuthCount = 0
		usr.FailedAuthAt = time.Time{}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "Update")
	}
	return nil
}

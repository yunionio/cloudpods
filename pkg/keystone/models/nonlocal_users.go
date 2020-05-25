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
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

// +onecloud:swagger-gen-ignore
type SNonlocalUserManager struct {
	db.SModelBaseManager
}

var NonlocalUserManager *SNonlocalUserManager

func init() {
	NonlocalUserManager = &SNonlocalUserManager{
		SModelBaseManager: db.NewModelBaseManager(
			SNonlocalUser{},
			"nonlocal_user",
			"nonlocal_user",
			"nonlocal_users",
		),
	}
	NonlocalUserManager.SetVirtualObject(NonlocalUserManager)
}

/*
+-----------+--------------+------+-----+---------+-------+
| Field     | Type         | Null | Key | Default | Extra |
+-----------+--------------+------+-----+---------+-------+
| domain_id | varchar(64)  | NO   | PRI | NULL    |       |
| name      | varchar(255) | NO   | PRI | NULL    |       |
| user_id   | varchar(64)  | NO   | UNI | NULL    |       |
+-----------+--------------+------+-----+---------+-------+
*/

type SNonlocalUser struct {
	db.SModelBase

	DomainId string `width:"64" charset:"ascii" primary:"true"`
	Name     string `width:"255" charset:"utf8" primary:"true"`
	UserId   string `width:"64" charset:"ascii" nullable:"false" index:"true"`
}

/*
func (manager *SNonlocalUserManager) Register(ctx context.Context, domainId string, name string) (*SNonlocalUser, error) {
	key := fmt.Sprintf("%s-%s", domainId, name)
	lockman.LockRawObject(ctx, manager.Keyword(), key)
	defer lockman.ReleaseRawObject(ctx, manager.Keyword(), key)

	obj, err := db.NewModelObject(manager)
	if err != nil {
		return nil, errors.Wrap(err, "NewModelObject")
	}
	nonlocalUser := obj.(*SNonlocalUser)
	q := manager.Query().Equals("domain_id", domainId).Equals("name", name)
	err = q.First(nonlocalUser)
	if err == nil {
		return nonlocalUser, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "Query")
	}

	pubId, err := IdmappingManager.registerIdMap(ctx, domainId, name, api.IdMappingEntityUser)
	if err != nil {
		return nil, errors.Wrap(err, "IdmappingManager.registerIdMap")
	}

	nonlocalUser.UserId = pubId
	nonlocalUser.Name = name
	nonlocalUser.DomainId = domainId

	err = manager.TableSpec().Insert(nonlocalUser)
	if err != nil {
		return nil, errors.Wrap(err, "Insert")
	}

	return nonlocalUser, nil
}
*/

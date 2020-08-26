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

import "yunion.io/x/onecloud/pkg/cloudcommon/db"

// +onecloud:swagger-gen-ignore
type SFederatedUserManager struct {
	db.SResourceBaseManager
}

var (
	FederatedUserManager *SFederatedUserManager
)

func init() {
	FederatedUserManager = &SFederatedUserManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SFederatedUser{},
			"federated_user",
			"federated_user",
			"federated_users",
		),
	}
	FederatedUserManager.SetVirtualObject(FederatedUserManager)
}

/*
desc federated_user;
+--------------+--------------+------+-----+---------+----------------+
| Field        | Type         | Null | Key | Default | Extra          |
+--------------+--------------+------+-----+---------+----------------+
| id           | int(11)      | NO   | PRI | NULL    | auto_increment |
| user_id      | varchar(64)  | NO   | MUL | NULL    |                |
| idp_id       | varchar(64)  | NO   | MUL | NULL    |                |
| protocol_id  | varchar(64)  | NO   | MUL | NULL    |                |
| unique_id    | varchar(255) | NO   |     | NULL    |                |
| display_name | varchar(255) | YES  |     | NULL    |                |
+--------------+--------------+------+-----+---------+----------------+
*/

type SFederatedUser struct {
	db.SResourceBase

	Id          int    `nullable:"false" primary:"true" auto_increment:"true"`
	UserId      string `width:"64" charset:"ascii" nullable:"false" index:"true"`
	IdpId       string `width:"64" charset:"ascii" nullable:"false" index:"true"`
	ProtocolId  string `width:"64" charset:"ascii" nullable:"false" index:"true"`
	UniqueId    string `width:"255" charset:"ascii" nullable:"false"`
	DisplayName string `width:"255" charset:"utf8" nullable:"true"`
}

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
type SImpliedRoleManager struct {
	db.SModelBaseManager
}

var (
	ImpliedRoleManager *SImpliedRoleManager
)

func init() {
	ImpliedRoleManager = &SImpliedRoleManager{
		SModelBaseManager: db.NewModelBaseManager(
			SImpliedRole{},
			"implied_role",
			"implied_role",
			"implied_roles",
		),
	}
	ImpliedRoleManager.SetVirtualObject(ImpliedRoleManager)
}

/*
desc implied_role;
+-----------------+-------------+------+-----+---------+-------+
| Field           | Type        | Null | Key | Default | Extra |
+-----------------+-------------+------+-----+---------+-------+
| prior_role_id   | varchar(64) | NO   | PRI | NULL    |       |
| implied_role_id | varchar(64) | NO   | PRI | NULL    |       |
+-----------------+-------------+------+-----+---------+-------+
*/

type SImpliedRole struct {
	db.SModelBase

	PriorRoleId   string `width:"64" charset:"ascii" nullable:"false" primary:"true"`
	ImpliedRoleId string `width:"64" charset:"ascii" nullable:"false" primary:"true"`
}

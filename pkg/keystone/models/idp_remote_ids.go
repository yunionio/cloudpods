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
type SIdpRemoteIdsManager struct {
	db.SModelBaseManager
}

var (
	IdpRemoteIdsManager *SIdpRemoteIdsManager
)

func init() {
	IdpRemoteIdsManager = &SIdpRemoteIdsManager{
		SModelBaseManager: db.NewModelBaseManager(
			SIdpRemoteIds{},
			"idp_remote_ids",
			"idp_remote_ids",
			"idp_remote_ids",
		),
	}
	IdpRemoteIdsManager.SetVirtualObject(IdpRemoteIdsManager)
}

/*
desc idp_remote_ids;
+-----------+--------------+------+-----+---------+-------+
| Field     | Type         | Null | Key | Default | Extra |
+-----------+--------------+------+-----+---------+-------+
| idp_id    | varchar(64)  | YES  | MUL | NULL    |       |
| remote_id | varchar(255) | NO   | PRI | NULL    |       |
+-----------+--------------+------+-----+---------+-------+
*/

type SIdpRemoteIds struct {
	db.SModelBase

	IdpId    string `width:"64" charset:"ascii" nullable:"true"`
	RemoteId string `width:"255" charset:"ascii" nullable:"false" primary:"true"`
}

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
type SFederationProtocolManager struct {
	db.SModelBaseManager
}

var (
	FederationProtocolManager *SFederationProtocolManager
)

func init() {
	FederationProtocolManager = &SFederationProtocolManager{
		SModelBaseManager: db.NewModelBaseManager(
			SFederationProtocol{},
			"federation_protocol",
			"federation_protocol",
			"federation_protocols",
		),
	}
	FederationProtocolManager.SetVirtualObject(FederationProtocolManager)
}

/*
desc federation_protocol;
+------------+-------------+------+-----+---------+-------+
| Field      | Type        | Null | Key | Default | Extra |
+------------+-------------+------+-----+---------+-------+
| id         | varchar(64) | NO   | PRI | NULL    |       |
| idp_id     | varchar(64) | NO   | PRI | NULL    |       |
| mapping_id | varchar(64) | NO   |     | NULL    |       |
+------------+-------------+------+-----+---------+-------+
*/

type SFederationProtocol struct {
	db.SModelBase

	Id        string `width:"64" charset:"ascii" nullable:"false" primary:"true"`
	IdpId     string `width:"64" charset:"ascii" nullable:"false" primary:"true"`
	MappingId string `width:"64" charset:"ascii" nullable:"false"`
}

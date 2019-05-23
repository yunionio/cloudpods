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
	"yunion.io/x/pkg/tristate"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

type SIdentityProviderManager struct {
	db.SStandaloneResourceBaseManager
}

var (
	IdentityProviderManager *SIdentityProviderManager
)

func init() {
	IdentityProviderManager = &SIdentityProviderManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SIdentityProvider{},
			"identity_provider",
			"identity_provider",
			"identity_providers",
		),
	}
	IdentityProviderManager.SetVirtualObject(IdentityProviderManager)
}

/*
desc identity_provider;
+-------------+-------------+------+-----+---------+-------+
| Field       | Type        | Null | Key | Default | Extra |
+-------------+-------------+------+-----+---------+-------+
| id          | varchar(64) | NO   | PRI | NULL    |       |
| enabled     | tinyint(1)  | NO   |     | NULL    |       |
| description | text        | YES  |     | NULL    |       |
| domain_id   | varchar(64) | NO   | MUL | NULL    |       |
+-------------+-------------+------+-----+---------+-------+
*/

type SIdentityProvider struct {
	db.SStandaloneResourceBase

	Enabled  tristate.TriState `nullable:"false" default:"true"`
	DomainId string            `width:"64" charset:"ascii" nullable:"false" index:"true"`
}

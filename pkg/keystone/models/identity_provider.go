package models

import (
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/pkg/tristate"
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

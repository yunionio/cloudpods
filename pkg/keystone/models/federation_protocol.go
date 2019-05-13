package models

import "yunion.io/x/onecloud/pkg/cloudcommon/db"

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

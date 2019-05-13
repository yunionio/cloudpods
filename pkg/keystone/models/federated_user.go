package models

import "yunion.io/x/onecloud/pkg/cloudcommon/db"

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

package models

import "yunion.io/x/onecloud/pkg/cloudcommon/db"

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

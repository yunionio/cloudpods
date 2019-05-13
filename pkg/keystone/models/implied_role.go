package models

import "yunion.io/x/onecloud/pkg/cloudcommon/db"

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

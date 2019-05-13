package models

import "yunion.io/x/onecloud/pkg/cloudcommon/db"

type SUserOptionManager struct {
	db.SModelBaseManager
}

var (
	UserOptionManager *SUserOptionManager
)

func init() {
	UserOptionManager = &SUserOptionManager{
		SModelBaseManager: db.NewModelBaseManager(
			SUserOption{},
			"user_option",
			"user_option",
			"user_options",
		),
	}
}

/*
desc user_option;
+--------------+-------------+------+-----+---------+-------+
| Field        | Type        | Null | Key | Default | Extra |
+--------------+-------------+------+-----+---------+-------+
| user_id      | varchar(64) | NO   | PRI | NULL    |       |
| option_id    | varchar(4)  | NO   | PRI | NULL    |       |
| option_value | text        | YES  |     | NULL    |       |
+--------------+-------------+------+-----+---------+-------+
*/

type SUserOption struct {
	db.SModelBase

	UserId      string `width:"64" charset:"ascii" nullable:"false" primary:"true"`
	OptionId    string `width:"4" charset:"ascii" nullable:"false" primary:"true"`
	OptionValue string `nullable:"true"`
}

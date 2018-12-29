package models

import "yunion.io/x/onecloud/pkg/cloudcommon/db"

type SImageMemberManager struct {
	db.SResourceBaseManager
}

var ImageMemberManager *SImageMemberManager

func init() {
	ImageMemberManager = &SImageMemberManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SImageMember{},
			"image_members",
			"image_member",
			"image_members",
		),
	}

	ImageMemberManager.TableSpec().AddIndex(true, "image_id", "member")
}

/*
+------------+--------------+------+-----+---------+----------------+
| Field      | Type         | Null | Key | Default | Extra          |
+------------+--------------+------+-----+---------+----------------+
| id         | int(11)      | NO   | PRI | NULL    | auto_increment |
| image_id   | varchar(36)  | NO   | MUL | NULL    |                |
| member     | varchar(255) | NO   |     | NULL    |                |
| can_share  | tinyint(1)   | NO   |     | NULL    |                |
| created_at | datetime     | NO   |     | NULL    |                |
| updated_at | datetime     | YES  |     | NULL    |                |
| deleted_at | datetime     | YES  |     | NULL    |                |
| deleted    | tinyint(1)   | NO   | MUL | NULL    |                |
+------------+--------------+------+-----+---------+----------------+
*/

type SImageMember struct {
	SImagePeripheral

	Member   string `width:"255" nullable:"false"`
	CanShare bool   `nullable:"false"`
}

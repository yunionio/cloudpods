package models

import "yunion.io/x/onecloud/pkg/cloudcommon/db"

type SImageTagManager struct {
	db.SResourceBaseManager
}

var ImageTagManager *SImageTagManager

func init() {
	ImageTagManager = &SImageTagManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SImageTag{},
			"image_tags",
			"image_tag",
			"image_tags",
		),
	}
	ImageTagManager.TableSpec().AddIndex(true, "image_id", "value")
}

/*
+------------+--------------+------+-----+---------+----------------+
| Field      | Type         | Null | Key | Default | Extra          |
+------------+--------------+------+-----+---------+----------------+
| id         | int(11)      | NO   | PRI | NULL    | auto_increment |
| image_id   | varchar(36)  | NO   | MUL | NULL    |                |
| value      | varchar(255) | NO   |     | NULL    |                |
| created_at | datetime     | NO   |     | NULL    |                |
| updated_at | datetime     | YES  |     | NULL    |                |
| deleted_at | datetime     | YES  |     | NULL    |                |
| deleted    | tinyint(1)   | NO   |     | NULL    |                |
+------------+--------------+------+-----+---------+----------------+
*/
type SImageTag struct {
	SImagePeripheral

	Value string `width:"255" nullable:"false"`
}

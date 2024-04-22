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
	ImageMemberManager.SetVirtualObject(ImageMemberManager)

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
// +onecloud:swagger-gen-ignore
type SImageMember struct {
	SImagePeripheral

	Member   string `width:"255" nullable:"false"`
	CanShare bool   `nullable:"false"`
}

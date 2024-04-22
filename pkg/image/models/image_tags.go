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
	ImageTagManager.SetVirtualObject(ImageTagManager)
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
// +onecloud:swagger-gen-ignore
type SImageTag struct {
	SImagePeripheral

	Value string `width:"255" nullable:"false"`
}

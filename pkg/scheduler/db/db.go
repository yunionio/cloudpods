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

package db

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"

	"yunion.io/x/log"
)

var DB *gorm.DB

func Init(dialect string, args ...interface{}) error {
	if DB == nil {
		db, err := gorm.Open(dialect, args...)
		if err != nil {
			return err
		}
		DB = db
		return nil
	}
	log.Warningf("DB: %s , Conn: %v already connected...", dialect, args)
	return nil
}

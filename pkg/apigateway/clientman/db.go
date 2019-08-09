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

package clientman

import (
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/pkg/errors"

	"yunion.io/x/log"
)

var DB *gorm.DB

type TokenRecord struct {
	ID       uint   `gorm:"primary_key"`
	TokenID  string `gorm:"not null;unique"`
	Token    string `gorm:"not null;unique"`
	TokenStr string `gorm:"not null;DEFAULT:''"`
	Totp     string `gorm:"not null;DEFAULT:''"`
	Version  string `gorm:"not null;DEFAULT:'v2'"`
	ExpireAt time.Time
}

func initDB(dialect string, args ...interface{}) error {
	if DB == nil {
		db, err := gorm.Open(dialect, args...)
		if err != nil {
			return errors.Wrapf(err, "Init DB by args: %v", args)
		}
		// Migrate the schema
		db.AutoMigrate(&TokenRecord{})
		DB = db
		return nil
	}
	log.Warningf("DB: %s , Conn: %v already connected...", dialect, args)
	return nil
}

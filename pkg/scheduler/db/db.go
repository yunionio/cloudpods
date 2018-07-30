package db

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"

	"github.com/yunionio/log"
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

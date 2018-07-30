package sqlchemy

import (
	"database/sql"

	"github.com/yunionio/log"
)

var _db *sql.DB

func SetDB(db *sql.DB) {
	_db = db
}

func GetDB() *sql.DB {
	return _db
}

func CloseDB() {
	_db.Close()
	_db = nil
}

type tableName struct {
	Name string
}

func GetTables() []string {
	tables := make([]tableName, 0)
	q := NewRawQuery("SHOW TABLES", "name")
	err := q.All(&tables)
	if err != nil {
		log.Errorf("show tables fail %s", err)
		return nil
	}
	ret := make([]string, len(tables))
	for i, t := range tables {
		ret[i] = t.Name
	}
	return ret
}

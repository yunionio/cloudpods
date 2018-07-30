package sqlchemy

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/yunionio/log"
	"github.com/yunionio/pkg/gotypes"
	"github.com/yunionio/pkg/util/reflectutils"
)

func (t *STableSpec) Insert(dt interface{}) error {
	beforeInsertFunc := reflect.ValueOf(dt).MethodByName("BeforeInsert")
	if beforeInsertFunc.IsValid() && !beforeInsertFunc.IsNil() {
		beforeInsertFunc.Call([]reflect.Value{})
	}

	// dataType := reflect.TypeOf(dt).Elem()
	dataValue := reflect.ValueOf(dt).Elem()

	var autoIncField string
	createdAtFields := make([]string, 0)

	names := make([]string, 0)
	format := make([]string, 0)
	values := make([]interface{}, 0)
	fields := reflectutils.FetchStructFieldNameValueInterfaces(dataValue)
	for _, c := range t.columns {
		isAutoInc := false
		nc, ok := c.(*SIntegerColumn)
		if ok && nc.IsAutoIncrement {
			isAutoInc = true
		}

		k := c.Name()

		dtc, ok := c.(*SDateTimeColumn)
		ov := fields[k]

		// log.Debugf("field %s value %s %s", k, ov, ov==nil)
		if ok && (dtc.IsCreatedAt || dtc.IsUpdatedAt) {
			createdAtFields = append(createdAtFields, k)
			names = append(names, fmt.Sprintf("`%s`", k))
			format = append(format, "NOW()")
		} else if ov != nil && !c.IsZero(ov) && !isAutoInc {
			v := c.ConvertFromValue(ov)
			values = append(values, v)
			names = append(names, fmt.Sprintf("`%s`", k))
			format = append(format, "?")
		} else if c.IsPrimary() {
			if isAutoInc {
				if len(autoIncField) == 0 {
					autoIncField = k
				} else {
					log.Fatalf("multiple auto_increment columns???")
				}
			} else {
				return fmt.Errorf("fail to insert for null primary key `%s`", k)
			}
		}
	}
	insertSql := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES(%s)",
		t.name,
		strings.Join(names, ", "),
		strings.Join(format, ", "))
	if DEBUG_SQLCHEMY {
		log.Debugf("%s values: %s", insertSql, values)
	}
	results, err := _db.Exec(insertSql, values...)
	if err != nil {
		return err
	}
	affectCnt, err := results.RowsAffected()
	if err != nil {
		return err
	}
	if affectCnt != 1 {
		return fmt.Errorf("Insert affected cnt %d != 1", affectCnt)
	}
	if len(autoIncField) > 0 {
		lastId, err := results.LastInsertId()
		if err == nil {
			val, ok := reflectutils.FindStructFieldValue(dataValue, autoIncField)
			if ok {
				gotypes.SetValue(val, fmt.Sprint(lastId))
			}
		}
	}
	return nil
}

package sqlchemy

import (
	"fmt"
	"reflect"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/reflectutils"
)

func (t *STableSpec) Insert(dt interface{}) error {
	return t.insert(dt, false)
}

func (t *STableSpec) insertSqlPrep(dataFields reflectutils.SStructFieldValueSet) (string, []interface{}, error) {
	var autoIncField string
	createdAtFields := make([]string, 0)

	names := make([]string, 0)
	format := make([]string, 0)
	values := make([]interface{}, 0)

	for _, c := range t.columns {
		isAutoInc := false
		nc, ok := c.(*SIntegerColumn)
		if ok && nc.IsAutoIncrement {
			isAutoInc = true
		}

		k := c.Name()

		dtc, ok := c.(*SDateTimeColumn)
		ov, find := dataFields.GetInterface(k)

		if !find {
			continue
		}

		if ok && (dtc.IsCreatedAt || dtc.IsUpdatedAt) {
			createdAtFields = append(createdAtFields, k)
			names = append(names, fmt.Sprintf("`%s`", k))
			if c.IsZero(ov) {
				format = append(format, "UTC_TIMESTAMP()")
			} else {
				values = append(values, ov)
				format = append(format, "?")
			}
		} else if c.IsSupportDefault() && len(c.Default()) > 0 && !gotypes.IsNil(ov) && c.IsZero(ov) { // empty text value
			val := c.ConvertFromString(c.Default())
			values = append(values, val)
			names = append(names, fmt.Sprintf("`%s`", k))
			format = append(format, "?")
		} else if !gotypes.IsNil(ov) && (!c.IsZero(ov) || (!c.IsPointer() && !c.IsText())) && !isAutoInc {
			v := c.ConvertFromValue(ov)
			values = append(values, v)
			names = append(names, fmt.Sprintf("`%s`", k))
			format = append(format, "?")
		} else if c.IsPrimary() {
			if isAutoInc {
				if len(autoIncField) > 0 {
					panic(fmt.Sprintf("multiple auto_increment columns: %q, %q", autoIncField, k))
				}
				autoIncField = k
			} else {
				return "", nil, fmt.Errorf("cannot insert for null primary key %q", k)
			}
		}
	}

	insertSql := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES(%s)",
		t.name,
		strings.Join(names, ", "),
		strings.Join(format, ", "))
	return insertSql, values, nil
}

func (t *STableSpec) insert(data interface{}, debug bool) error {
	beforeInsertFunc := reflect.ValueOf(data).MethodByName("BeforeInsert")
	if beforeInsertFunc.IsValid() && !beforeInsertFunc.IsNil() {
		beforeInsertFunc.Call([]reflect.Value{})
	}

	dataValue := reflect.ValueOf(data).Elem()
	dataFields := reflectutils.FetchStructFieldValueSet(dataValue)
	insertSql, values, err := t.insertSqlPrep(dataFields)
	if err != nil {
		return err
	}

	if DEBUG_SQLCHEMY || debug {
		log.Debugf("%s values: %v", insertSql, values)
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

	/*
		if len(autoIncField) > 0 {
			lastId, err := results.LastInsertId()
			if err == nil {
				val, ok := reflectutils.FindStructFieldValue(dataValue, autoIncField)
				if ok {
					gotypes.SetValue(val, fmt.Sprint(lastId))
				}
			}
		}
	*/

	// query the value, so default value can be feedback into the object
	// fields = reflectutils.FetchStructFieldNameValueInterfaces(dataValue)
	q := t.Query()
	for _, c := range t.columns {
		if c.IsPrimary() {
			nc, ok := c.(*SIntegerColumn)
			if ok && nc.IsAutoIncrement {
				lastId, err := results.LastInsertId()
				if err != nil {
					err := fmt.Errorf("fetching lastInsertId failed: %v", err)
					return err
				} else {
					q = q.Equals(c.Name(), lastId)
				}
			} else {
				priVal, _ := dataFields.GetInterface(c.Name())
				if !gotypes.IsNil(priVal) {
					q = q.Equals(c.Name(), priVal)
				}
			}
		}
	}
	err = q.First(data)
	if err != nil {
		err := fmt.Errorf("query after insert failed: %v", err)
		return err
	}

	return nil
}

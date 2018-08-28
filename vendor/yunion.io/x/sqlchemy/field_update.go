package sqlchemy

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"

	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/reflectutils"
)

func (ts *STableSpec) GetUpdateColumnValue(dataType reflect.Type, dataValue reflect.Value, cv map[string]interface{}, fields map[string]interface{}) error {
	for i := 0; i < dataType.NumField(); i++ {
		fieldType := dataType.Field(i)
		if gotypes.IsFieldExportable(fieldType.Name) {
			fieldValue := dataValue.Field(i)
			newValue, ok := fields[fieldType.Name]
			if ok && fieldType.Anonymous {
				return errors.New("Unsupported update anonymous field")
			}
			if ok {
				columnName := reflectutils.GetStructFieldName(&fieldType)
				cv[columnName] = newValue
				continue
			}
			if fieldType.Anonymous {
				err := ts.GetUpdateColumnValue(fieldType.Type, fieldValue, cv, fields)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (ts *STableSpec) UpdateFields(dt interface{}, fields map[string]interface{}) error {
	return ts.updateFields(dt, fields, false)
}

// params dt: model struct, fileds: {struct-field-name-string: update-value}
// find primary key and index key
// find fields correlatively columns
// joint sql and executed
func (ts *STableSpec) updateFields(dt interface{}, fields map[string]interface{}, debug bool) error {
	dataValue := reflect.ValueOf(dt)
	if dataValue.Kind() == reflect.Ptr {
		dataValue = dataValue.Elem()
	}

	//cv: {"column name": "update value"}
	cv := make(map[string]interface{}, 0)
	dataType := dataValue.Type()
	ts.GetUpdateColumnValue(dataType, dataValue, cv, fields)

	fullFields := reflectutils.FetchStructFieldNameValueInterfaces(dataValue)
	versionFields := make([]string, 0)
	updatedFields := make([]string, 0)
	primaryCols := make(map[string]interface{}, 0)
	indexCols := make(map[string]interface{}, 0)
	for _, col := range ts.Columns() {
		name := col.Name()
		colValue := fullFields[name]
		if col.IsPrimary() && !col.IsZero(colValue) {
			primaryCols[name] = colValue
			continue
		} else if col.IsKeyIndex() && !col.IsZero(colValue) {
			indexCols[name] = colValue
			continue
		}
		intCol, ok := col.(*SIntegerColumn)
		if ok && intCol.IsAutoVersion {
			versionFields = append(versionFields, name)
			continue
		}
		dateCol, ok := col.(*SDateTimeColumn)
		if ok && dateCol.IsUpdatedAt {
			updatedFields = append(updatedFields, name)
			continue
		}
		if _, exist := cv[name]; exist {
			cv[name] = col.ConvertFromValue(cv[name])
		}
	}

	vars := make([]interface{}, 0)
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("UPDATE `%s` SET ", ts.name))
	first := true
	for k, v := range cv {
		if first {
			first = false
		} else {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("`%s` = ?", k))
		vars = append(vars, v)
	}
	for _, versionField := range versionFields {
		buf.WriteString(fmt.Sprintf(", `%s` = `%s` + 1", versionField, versionField))
	}
	for _, updatedField := range updatedFields {
		buf.WriteString(fmt.Sprintf(", `%s` = UTC_TIMESTAMP()", updatedField))
	}
	buf.WriteString(" WHERE ")
	first = true
	var indexFilter map[string]interface{}
	if len(primaryCols) > 0 {
		indexFilter = primaryCols
	} else if len(indexCols) > 0 {
		indexFilter = indexCols
	} else {
		return fmt.Errorf("neither primary key nor key indexes empty???")
	}

	for k, v := range indexFilter {
		if first {
			first = false
		} else {
			buf.WriteString(" AND ")
		}
		buf.WriteString(fmt.Sprintf("`%s` = ?", k))
		vars = append(vars, v)
	}

	if DEBUG_SQLCHEMY || debug {
		log.Infof("Update: %s", buf.String())
	}
	results, err := _db.Exec(buf.String(), vars...)
	if err != nil {
		return err
	}
	aCnt, err := results.RowsAffected()
	if err != nil {
		return err
	}
	if aCnt != 1 {
		return fmt.Errorf("affected rows %d != 1", aCnt)
	}
	return nil
}

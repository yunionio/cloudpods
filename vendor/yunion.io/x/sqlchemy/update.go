package sqlchemy

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/pkg/utils"
)

type SUpdateSession struct {
	oValue    reflect.Value
	tableSpec *STableSpec
}

func (ts *STableSpec) prepareUpdate(dt interface{}) (*SUpdateSession, error) {
	if reflect.ValueOf(dt).Kind() != reflect.Ptr {
		return nil, fmt.Errorf("Update input must be a Pointer")
	}
	dataValue := reflect.ValueOf(dt).Elem()
	fields := reflectutils.FetchStructFieldNameValueInterfaces(dataValue) //  fetchStructFieldNameValue(dataType, dataValue)

	zeroPrimary := make([]string, 0)
	zeroKeyIndex := make([]string, 0)
	for _, c := range ts.columns {
		k := c.Name()
		ov, ok := fields[k]
		if !ok {
			continue
		}
		if c.IsPrimary() && c.IsZero(ov) {
			zeroPrimary = append(zeroPrimary, k)
		} else if c.IsKeyIndex() && c.IsZero(ov) {
			zeroKeyIndex = append(zeroKeyIndex, k)
		}
	}

	if len(zeroPrimary) > 0 && len(zeroKeyIndex) > 0 {
		return nil, fmt.Errorf("not a valid data, primary key %s and key index %s are empty",
			strings.Join(zeroPrimary, ","), strings.Join(zeroKeyIndex, ","))
	}

	originValue := gotypes.DeepCopyRv(dataValue)
	us := SUpdateSession{oValue: originValue, tableSpec: ts}
	return &us, nil
}

type SUpdateDiff struct {
	old interface{}
	new interface{}
	col IColumnSpec
}

func UpdateDiffString(diff map[string]SUpdateDiff) string {
	items := make([]string, 0)
	for k, v := range diff {
		items = append(items, fmt.Sprintf("%s: %s -> %s", k,
			utils.TruncateString(v.old, 32),
			utils.TruncateString(v.new, 32)))
	}
	return strings.Join(items, "; ")
}

func (us *SUpdateSession) saveUpdate(dt interface{}) (map[string]SUpdateDiff, error) {
	beforeUpdateFunc := reflect.ValueOf(dt).MethodByName("BeforeUpdate")
	if beforeUpdateFunc.IsValid() && !beforeUpdateFunc.IsNil() {
		beforeUpdateFunc.Call([]reflect.Value{})
	}

	// dataType := reflect.TypeOf(dt).Elem()
	dataValue := reflect.ValueOf(dt).Elem()
	ofields := reflectutils.FetchStructFieldNameValueInterfaces(us.oValue)
	fields := reflectutils.FetchStructFieldNameValueInterfaces(dataValue)

	versionFields := make([]string, 0)
	updatedFields := make([]string, 0)
	primaries := make(map[string]interface{})
	keyIndexes := make(map[string]interface{})
	setters := make(map[string]SUpdateDiff)
	for _, c := range us.tableSpec.columns {
		k := c.Name()
		of := ofields[k]
		nf := fields[k]
		if !gotypes.IsNil(of) {
			if c.IsPrimary() && !c.IsZero(of) { // skip update primary key
				primaries[k] = of
				continue
			} else if c.IsKeyIndex() && !c.IsZero(of) {
				keyIndexes[k] = of
				continue
			}
		}
		nc, ok := c.(*SIntegerColumn)
		if ok && nc.IsAutoVersion {
			versionFields = append(versionFields, k)
			continue
		}
		dtc, ok := c.(*SDateTimeColumn)
		if ok && dtc.IsUpdatedAt {
			updatedFields = append(updatedFields, k)
			continue
		}
		if reflect.DeepEqual(of, nf) {
			continue
		}
		if c.IsZero(nf) && c.IsText() {
			nf = nil
		}
		setters[k] = SUpdateDiff{old: of, new: nf, col: c}
	}

	if len(setters) == 0 {
		return nil, ErrNoDataToUpdate
	}

	vars := make([]interface{}, 0)
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("UPDATE `%s` SET ", us.tableSpec.name))
	first := true
	for k, v := range setters {
		if first {
			first = false
		} else {
			buf.WriteString(", ")
		}
		if gotypes.IsNil(v.new) {
			buf.WriteString(fmt.Sprintf("`%s` = NULL", k))
		} else {
			buf.WriteString(fmt.Sprintf("`%s` = ?", k))
			vars = append(vars, v.col.ConvertFromValue(v.new))
		}
	}
	for _, versionField := range versionFields {
		buf.WriteString(fmt.Sprintf(", `%s` = `%s` + 1", versionField, versionField))
	}
	for _, updatedField := range updatedFields {
		buf.WriteString(fmt.Sprintf(", `%s` = UTC_TIMESTAMP()", updatedField))
	}
	buf.WriteString(" WHERE ")
	first = true
	var indexFields map[string]interface{}
	if len(primaries) > 0 {
		indexFields = primaries
	} else if len(keyIndexes) > 0 {
		indexFields = keyIndexes
	} else {
		return nil, fmt.Errorf("neither primary key nor key indexes empty???")
	}
	for k, v := range indexFields {
		if first {
			first = false
		} else {
			buf.WriteString(" AND ")
		}
		buf.WriteString(fmt.Sprintf("`%s` = ?", k))
		vars = append(vars, v)
	}

	if DEBUG_SQLCHEMY {
		log.Infof("Update: %s", buf.String())
	}
	results, err := _db.Exec(buf.String(), vars...)
	if err != nil {
		return nil, err
	}
	aCnt, err := results.RowsAffected()
	if err != nil {
		return nil, err
	}
	if aCnt != 1 {
		return nil, fmt.Errorf("affected rows %d != 1", aCnt)
	}
	return setters, nil
}

func (ts *STableSpec) Update(dt interface{}, doUpdate func() error) (map[string]SUpdateDiff, error) {
	session, err := ts.prepareUpdate(dt)
	if err != nil {
		return nil, err
	}
	err = doUpdate()
	if err != nil {
		return nil, err
	}
	diff, err := session.saveUpdate(dt)
	if err == ErrNoDataToUpdate {
		return nil, nil
	} else if err == nil {
		if DEBUG_SQLCHEMY {
			log.Debugf("Update diff: %s", UpdateDiffString(diff))
		}
	}
	return diff, err
}

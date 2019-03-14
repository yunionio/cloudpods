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
	fields := reflectutils.FetchStructFieldValueSet(dataValue) //  fetchStructFieldNameValue(dataType, dataValue)

	zeroPrimary := make([]string, 0)
	for _, c := range ts.columns {
		k := c.Name()
		ov, ok := fields.GetInterface(k)
		if !ok {
			continue
		}
		if c.IsPrimary() && c.IsZero(ov) {
			zeroPrimary = append(zeroPrimary, k)
		}
	}

	if len(zeroPrimary) > 0 {
		return nil, fmt.Errorf("not a valid data, primary key %s empty",
			strings.Join(zeroPrimary, ","))
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

type UpdateDiffs map[string]SUpdateDiff

func (uds UpdateDiffs) String() string {
	items := make([]string, 0, len(uds))
	for k, v := range uds {
		items = append(items, fmt.Sprintf("%s: %s -> %s", k,
			utils.TruncateString(v.old, 32),
			utils.TruncateString(v.new, 32)))
	}
	return strings.Join(items, "; ")
}

func (us *SUpdateSession) saveUpdate(dt interface{}) (UpdateDiffs, error) {
	beforeUpdateFunc := reflect.ValueOf(dt).MethodByName("BeforeUpdate")
	if beforeUpdateFunc.IsValid() && !beforeUpdateFunc.IsNil() {
		beforeUpdateFunc.Call([]reflect.Value{})
	}

	// dataType := reflect.TypeOf(dt).Elem()
	dataValue := reflect.ValueOf(dt).Elem()
	ofields := reflectutils.FetchStructFieldValueSet(us.oValue)
	fields := reflectutils.FetchStructFieldValueSet(dataValue)

	versionFields := make([]string, 0)
	updatedFields := make([]string, 0)
	primaries := make(map[string]interface{})
	setters := UpdateDiffs{}
	for _, c := range us.tableSpec.columns {
		k := c.Name()
		of, _ := ofields.GetInterface(k)
		nf, _ := fields.GetInterface(k)
		if !gotypes.IsNil(of) {
			if c.IsPrimary() && !c.IsZero(of) { // skip update primary key
				primaries[k] = of
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
	if len(primaries) == 0 {
		return nil, fmt.Errorf("primary key empty???")
	}
	for k, v := range primaries {
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

func (ts *STableSpec) Update(dt interface{}, doUpdate func() error) (UpdateDiffs, error) {
	session, err := ts.prepareUpdate(dt)
	if err != nil {
		return nil, err
	}
	err = doUpdate()
	if err != nil {
		return nil, err
	}
	uds, err := session.saveUpdate(dt)
	if err == ErrNoDataToUpdate {
		return nil, nil
	} else if err == nil {
		if DEBUG_SQLCHEMY {
			log.Debugf("Update diff: %s", uds)
		}
	}
	return uds, err
}

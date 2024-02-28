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

package sqlchemy

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/pkg/util/timeutils"
)

const (
	sqlLineLimit = 100
)

func (t *STableSpec) InsertBatch(dataList []interface{}) error {
	qChar := t.Database().backend.QuoteChar()

	var sql string
	var fieldCount int
	{
		buffer := new(bytes.Buffer)

		buffer.WriteString("INSERT INTO ")
		buffer.WriteString(qChar)
		buffer.WriteString(t.Name())
		buffer.WriteString(qChar)
		buffer.WriteString(" (")
		headers := make([]string, 0)
		format := make([]string, 0)

		for _, col := range t.Columns() {
			if col.IsAutoIncrement() {
				continue
			}
			name := col.Name()
			headers = append(headers, fmt.Sprintf("%s%s%s", qChar, name, qChar))
			if col.IsCreatedAt() || col.IsUpdatedAt() {
				if t.Database().backend.SupportMixedInsertVariables() {
					format = append(format, t.Database().backend.CurrentUTCTimeStampString())
				} else {
					format = append(format, "?")
					fieldCount++
				}
				continue
			}
			format = append(format, "?")
			fieldCount++
		}

		buffer.WriteString(strings.Join(headers, ","))
		buffer.WriteString(") VALUES ")
		buffer.WriteString("(")
		buffer.WriteString(strings.Join(format, ","))
		buffer.WriteString(")")

		sql = buffer.String()

		if DEBUG_SQLCHEMY {
			log.Debugf("batchInsert SQL: %s", buffer.String())
		}
	}

	batchParams := make([][]interface{}, 0)

	now := timeutils.UtcNow()
	errs := make([]error, 0)
	for i := range dataList {
		v := dataList[i]

		var params []interface{}

		modelValue := reflect.Indirect(reflect.ValueOf(v))
		beforeInsert(modelValue)
		dataFields := reflectutils.FetchStructFieldValueSet(modelValue)

		for _, col := range t.Columns() {
			if col.IsAutoIncrement() {
				continue
			}
			if col.IsCreatedAt() || col.IsUpdatedAt() {
				if !t.Database().backend.SupportMixedInsertVariables() {
					params = append(params, now)
				}
				continue
			}
			ov, find := dataFields.GetInterface(col.Name())
			if !find || gotypes.IsNil(ov) || col.IsZero(ov) {
				// empty column
				if col.IsSupportDefault() && (len(col.Default()) > 0 || col.IsString()) {
					params = append(params, col.ConvertFromString(col.Default()))
				} else {
					params = append(params, nil)
				}
			} else {
				params = append(params, col.ConvertFromValue(ov))
			}
		}

		if len(params) != fieldCount {
			log.Errorf("expect %d got %d(%#v)", fieldCount, len(params), params)
		}

		batchParams = append(batchParams, params)
		if len(batchParams) >= sqlLineLimit || (i+1) == len(dataList) {
			results, err := t.Database().TxBatchExec(sql, batchParams)
			if err != nil {
				return errors.Wrap(err, "TxBatchExec")
			}
			for _, result := range results {
				if result.Error != nil {
					errs = append(errs, result.Error)
				}
			}
			if len(errs) != 0 {
				return errors.NewAggregate(errs)
			}
			batchParams = make([][]interface{}, 0)
		}
	}

	return nil
}

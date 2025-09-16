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
	"fmt"
	"reflect"
	"strings"

	"yunion.io/x/log"
)

func getSQLFilters(filter map[string]interface{}, qChar string) ([]string, []interface{}) {
	conds := make([]string, 0, len(filter))
	params := make([]interface{}, 0, len(filter))
	for k, v := range filter {
		if reflect.TypeOf(v).Kind() == reflect.Slice || reflect.TypeOf(v).Kind() == reflect.Array {
			value := reflect.ValueOf(v)
			if value.Len() == 0 {
				continue
			}
			arr := make([]string, value.Len())
			for i := 0; i < value.Len(); i++ {
				arr[i] = "?"
				params = append(params, value.Index(i).Interface())
			}
			conds = append(conds, fmt.Sprintf("%s%s%s in (%s)", qChar, k, qChar, strings.Join(arr, ", ")))
		} else {
			conds = append(conds, fmt.Sprintf("%s%s%s = ?", qChar, k, qChar))
			params = append(params, v)
		}
	}
	return conds, params
}

func (ts *STableSpec) DeleteFrom(filters map[string]interface{}) error {
	buf := strings.Builder{}

	qChar := ts.Database().backend.QuoteChar()

	buf.WriteString("DELETE FROM ")
	buf.WriteString(qChar)
	buf.WriteString(ts.Name())
	buf.WriteString(qChar)

	conds, params := getSQLFilters(filters, qChar)

	if len(conds) > 0 {
		buf.WriteString(" WHERE ")
		buf.WriteString(strings.Join(conds, " AND "))
	}

	if DEBUG_SQLCHEMY {
		log.Infof("Update: %s %s", buf.String(), params)
	}

	_, err := ts.Database().TxExec(buf.String(), params...)
	return err
}

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
	"strings"

	"yunion.io/x/log"
)

func (ts *STableSpec) UpdateBatch(data map[string]interface{}, filter map[string]interface{}) error {
	if len(data) <= 0 {
		return nil
	}

	qChar := ts.Database().backend.QuoteChar()

	params := make([]interface{}, 0, len(data))
	setter := make([]string, 0, len(data))
	for k, v := range data {
		col := ts.ColumnSpec(k)
		if col == nil {
			log.Warningf("UpdateBatch: column %s not found", k)
			continue
		}
		setter = append(setter, fmt.Sprintf("%s%s%s = ?", qChar, k, qChar))
		params = append(params, col.ConvertFromValue(v))
	}
	conds, condparams := ts.getSQLFilters(filter, qChar)
	params = append(params, condparams...)

	buf := strings.Builder{}

	buf.WriteString("UPDATE ")
	buf.WriteString(qChar)
	buf.WriteString(ts.Name())
	buf.WriteString(qChar)
	buf.WriteString(" SET ")
	buf.WriteString(strings.Join(setter, ", "))

	if len(conds) > 0 {
		buf.WriteString(" WHERE ")
		buf.WriteString(strings.Join(conds, " AND "))
	}

	if DEBUG_SQLCHEMY {
		log.Infof("UpdateBATCH: %s %s", buf.String(), params)
	}

	_, err := ts.Database().Exec(buf.String(), params...)
	return err
}

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

package ovnutil

import (
	"encoding/json"

	"yunion.io/x/pkg/errors"
)

type List struct {
	Headings ListHeadings
	Data     []ListDataRow
}
type ListHeadings []string
type ListDataRow []ListDataColumn
type ListDataColumn = interface{}

const (
	ErrColumnIndexOverrun = errors.Error("column index overrun")
)

func (h ListHeadings) GetByIndex(i int) (string, error) {
	if i < len(h) {
		return h[i], nil
	}
	return "", ErrColumnIndexOverrun
}

func UnmarshalJSON(data []byte, rows ITable) error {
	list := &List{}
	if err := json.Unmarshal(data, list); err != nil {
		return err
	}
	for _, row := range list.Data {
		r := rows.NewRow()
		for ci := range row {
			col := row[ci]
			colName, err := list.Headings.GetByIndex(ci)
			if err != nil {
				return err
			}
			if err := r.SetColumn(colName, col); err != nil {
				return err
			}
		}
	}
	return nil
}

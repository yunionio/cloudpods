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

import "strings"

type STableConstraint struct {
	name         string
	columns      []string
	foreignTable string
	foreignKeys  []string
}

func NewTableConstraint(name string, cols []string, foreignTable string, fcols []string) STableConstraint {
	return STableConstraint{
		name:         name,
		columns:      cols,
		foreignTable: foreignTable,
		foreignKeys:  fcols,
	}
}

func FetchColumns(match string) []string {
	ret := make([]string, 0)
	if len(match) > 0 {
		for _, part := range strings.Split(match, ",") {
			if part[len(part)-1] == ')' {
				part = part[:strings.LastIndexByte(part, '(')]
			}
			part = strings.Trim(part, " `")
			if len(part) > 0 {
				ret = append(ret, part)
			}
		}
	}
	return ret
}

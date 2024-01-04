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
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/reflectutils"
)

func (c *STableField) IsAscii() bool {
	return c.spec.IsAscii()
}

func isFieldRequireAscii(f IQueryField) bool {
	if gotypes.IsNil(f) {
		return false
	}
	if tableField, ok := f.(*STableField); ok {
		return tableField.IsAscii()
	}
	return false
}

func isVariableAscii(v interface{}) bool {
	if gotypes.IsNil(v) {
		return true
	}
	switch v.(type) {
	case IQueryField, *SQuery, *SSubQuery:
		return true
	default:
		vals := reflectutils.ExpandInterface(v)
		for _, val := range vals {
			if strVal, ok := val.(string); ok && len(strVal) > 0 {
				if !isPrintableAsciiString(strVal) {
					return false
				}
			} else if strVal, ok := val.(*string); ok && !gotypes.IsNil(strVal) && len(*strVal) > 0 {
				if !isPrintableAsciiString(*strVal) {
					return false
				}
			}
		}
	}
	return true
}

func isPrintableAscii(b byte) bool {
	if b >= 32 && b <= 126 {
		return true
	}
	return false
}

func isPrintableAsciiString(str string) bool {
	for _, b := range []byte(str) {
		if !isPrintableAscii(b) {
			return false
		}
	}
	return true
}

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

package db

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/sqlchemy"
)

type SPagingConfig struct {
	Order        sqlchemy.QueryOrderType
	MarkerFields []string
	DefaultLimit int
}

func decodePagingMarker(markString string) []string {
	if len(markString) > 2 && markString[0] == '[' && markString[len(markString)-1] == ']' {
		markJson, _ := jsonutils.ParseString(markString)
		if markJson != nil {
			if markArray, ok := markJson.(*jsonutils.JSONArray); ok {
				return markArray.GetStringArray()
			}
		}
	}
	if len(markString) > 0 {
		return []string{markString}
	}
	return []string{}
}

func encodePagingMarker(markers []string) string {
	switch len(markers) {
	case 0:
		return ""
	case 1:
		return markers[0]
	default:
		return jsonutils.Marshal(markers).String()
	}
}

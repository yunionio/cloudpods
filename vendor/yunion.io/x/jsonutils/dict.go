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

package jsonutils

import (
	"yunion.io/x/pkg/sortedmap"
)

func (dict *JSONDict) Update(json JSONObject) {
	dict2, ok := json.(*JSONDict)
	if !ok {
		return
	}
	dict.data = sortedmap.Merge(dict.data, dict2.data)
}

func (dict *JSONDict) UpdateDefault(json JSONObject) {
	dict2, ok := json.(*JSONDict)
	if !ok {
		return
	}
	dict.data = sortedmap.Merge(dict2.data, dict.data)
}

func Diff(a, b *JSONDict) (aNoB, aDiffB, aAndB, bNoA *JSONDict) {
	aNoB = NewDict()
	aDiffB = NewDict()
	aAndB = NewDict()
	bNoA = NewDict()

	var aData, bData sortedmap.SSortedMap
	aNoB.data, aData, bData, bNoA.data = sortedmap.Split(a.data, b.data)
	for _, k := range aData.Keys() {
		aVal, _ := aData.Get(k)
		bVal, _ := bData.Get(k)
		aJson := aVal.(JSONObject)
		bJson := bVal.(JSONObject)
		if !aJson.Equals(bJson) {
			aDiffB.Set(k, NewArray(aJson, bJson))
		} else {
			aAndB.Set(k, aJson)
		}
	}

	return
}

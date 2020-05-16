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

func (dict *JSONDict) Update(json JSONObject) {
	dict2, ok := json.(*JSONDict)
	if !ok {
		return
	}
	for k, v := range dict2.data {
		dict.data[k] = v
	}
}

func (dict *JSONDict) UpdateDefault(json JSONObject) {
	dict2, ok := json.(*JSONDict)
	if !ok {
		return
	}
	for k, v := range dict2.data {
		if _, ok := dict.data[k]; !ok {
			dict.data[k] = v
		}
	}
}

func Diff(a, b *JSONDict) (aNoB, aDiffB, aAndB, bNoA *JSONDict) {
	keysA := a.SortedKeys()
	keysB := b.SortedKeys()
	aNoB = NewDict()
	aDiffB = NewDict()
	aAndB = NewDict()
	bNoA = NewDict()

	i := 0
	j := 0
	for i < len(keysA) || j < len(keysB) {
		if i < len(keysA) && j < len(keysB) {
			keyA := keysA[i]
			keyB := keysB[j]
			if keyA > keyB {
				aNoB.data[keyA] = a.data[keyA]
				i += 1
			} else if keyA < keyB {
				bNoA.data[keyB] = b.data[keyB]
				j += 1
			} else {
				valA := a.data[keysA[i]].String()
				valB := b.data[keysB[i]].String()
				if valA != valB {
					aDiffB.data[keyA] = NewArray(a.data[keyA], b.data[keyB])
				} else {
					aAndB.data[keyA] = a.data[keyA]
				}
				i += 1
				j += 1
			}
		} else if i < len(keysA) {
			aNoB.data[keysA[i]] = a.data[keysA[i]]
			i = i + 1
		} else if j < len(keysB) {
			bNoA.data[keysB[j]] = b.data[keysB[j]]
			j = j + 1
		}
	}

	return
}

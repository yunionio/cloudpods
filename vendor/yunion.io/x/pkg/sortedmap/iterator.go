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

package sortedmap

type SSortedMapIterator struct {
	smap  SSortedMap
	index int
}

func (i *SSortedMapIterator) Init(smap SSortedMap) {
	i.smap = smap
	i.index = 0
}

func (i SSortedMapIterator) HasMore() bool {
	return i.index < len(i.smap)
}

func (i *SSortedMapIterator) Next() {
	i.index += 1
}

func (i SSortedMapIterator) Get() (string, interface{}) {
	return i.smap[i.index].key, i.smap[i.index].value
}

func NewIterator(smap SSortedMap) *SSortedMapIterator {
	iter := &SSortedMapIterator{}
	iter.Init(smap)
	return iter
}

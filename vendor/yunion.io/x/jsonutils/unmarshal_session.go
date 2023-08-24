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
	"reflect"
)

type sJsonNodeValues struct {
	nodeValue    reflect.Value
	nodeValueSet bool
	targetValues []reflect.Value
}

type sJsonUnmarshalSession struct {
	objectMap map[int]*sJsonNodeValues
}

func newJsonUnmarshalSession() *sJsonUnmarshalSession {
	return &sJsonUnmarshalSession{
		objectMap: make(map[int]*sJsonNodeValues),
	}
}

func (s *sJsonUnmarshalSession) saveNodeValue(nodeId int, val reflect.Value) {
	if nv, ok := s.objectMap[nodeId]; !ok {
		s.objectMap[nodeId] = &sJsonNodeValues{
			nodeValue:    val,
			nodeValueSet: true,
		}
	} else {
		nv.nodeValue = val
		nv.nodeValueSet = true
		for i := range nv.targetValues {
			nv.targetValues[i].Set(val)
		}
		nv.targetValues = nil
	}
}

func (s *sJsonUnmarshalSession) setPointerValue(nodeId int, val reflect.Value) error {
	if nv, ok := s.objectMap[nodeId]; ok && nv.nodeValueSet {
		val.Set(nv.nodeValue)
	} else if ok && !nv.nodeValueSet {
		nv.targetValues = append(nv.targetValues, val)
	} else {
		s.objectMap[nodeId] = &sJsonNodeValues{
			nodeValueSet: false,
			targetValues: []reflect.Value{
				val,
			},
		}
	}
	return nil
}
